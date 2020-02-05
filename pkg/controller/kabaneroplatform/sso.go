package kabaneroplatform

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/go-logr/logr"
	kabanerov1alpha2 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha2"

	mf "github.com/kabanero-io/manifestival"
	appsv1 "github.com/openshift/api/apps/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	sso_true string = "True"
	sso_false string = "False"
	sso_db_secret_name = "kabanero-sso-db-secret"
)

func reconcileSso(ctx context.Context, k *kabanerov1alpha2.Kabanero, c client.Client, reqLogger logr.Logger) error {
	if k.Spec.Sso.Enable == false {
		return disableSso(ctx, k, c, reqLogger)
	}

	// Figure out what version of the orchestration we are going to use.
	noOverrideVersion := ""
	rev, err := resolveSoftwareRevision(k, "sso", noOverrideVersion)
	if err != nil {
		return err
	}
	
	//The context which will be used to render any templates
	templateContext := make(map[string]interface{})

	// Go make sure that the necessary secret has been created.
	if len(k.Spec.Sso.AdminSecretName) == 0 {
		return errors.New("The SSO admin secret name must be specified in the Kabanero CR instance")
	}
	
	secretInstance := &corev1.Secret{}
	err = c.Get(context.Background(), types.NamespacedName{
		Name:      k.Spec.Sso.AdminSecretName,
		Namespace: k.ObjectMeta.Namespace}, secretInstance)

	if err != nil {
		if kerrors.IsNotFound(err) == false {
			return fmt.Errorf("The SSO admin secret was not found: %v", err.Error())
		}

		return fmt.Errorf("Could not retrieve the SSO admin secret: %v", err.Error())
	}

	// Make sure the required keys are assigned.
	ssoAdminUserName, ok := secretInstance.Data["username"]
	if (!ok) || (len(ssoAdminUserName) == 0) {
		return fmt.Errorf("The SSO admin secret %v does not contain key 'username'", k.Spec.Sso.AdminSecretName)
	}

	ssoAdminPassword, ok := secretInstance.Data["password"]
	if (!ok) || (len(ssoAdminPassword) == 0) {
		return fmt.Errorf("The SSO admin secret %v does not contain key 'password'", k.Spec.Sso.AdminSecretName)
	}

	ssoRealm, ok := secretInstance.Data["realm"]
	if (!ok) || (len(ssoRealm) == 0) {
		return fmt.Errorf("The SSO admin secret %v does not contain key 'realm'", k.Spec.Sso.AdminSecretName)
	}
	
	// Create DB secret if it does not exist
	err = createDbSecret(k, c, reqLogger)
	if err != nil {
		return fmt.Errorf("Failed to create the SSO DB secret: %v", err.Error())
	}
	
	templateContext["ssoAdminSecretName"] = k.Spec.Sso.AdminSecretName
	templateContext["ssoDbSecretName"] = sso_db_secret_name
	
	f, err := rev.OpenOrchestration("sso.yaml")
	if err != nil {
		return err
	}

	s, err := renderOrchestration(f, templateContext)
	if err != nil {
		return err
	}

	m, err := mf.FromReader(strings.NewReader(s), c)
	if err != nil {
		return err
	}

	transforms := []mf.Transformer{
		mf.InjectOwner(k),
		mf.InjectNamespace(k.GetNamespace()),
	}

	err = m.Transform(transforms...)
	if err != nil {
		return err
	}

	err = m.ApplyAll()
	if err != nil {
		return err
	}
	
	return nil
}

func disableSso(ctx context.Context, k *kabanerov1alpha2.Kabanero, c client.Client, reqLogger logr.Logger) error {
	// Figure out what version of the orchestration we are going to use.
	noOverrideVersion := ""
	rev, err := resolveSoftwareRevision(k, "sso", noOverrideVersion)
	if err != nil {
		return err
	}
	
	// The context which will be used to render any templates.  Note that
	// since we're just going to delete things, these values don't matter
	// to much.
	templateContext := make(map[string]interface{})
	templateContext["ssoAdminSecretName"] = "default"
	templateContext["ssoDbSecretName"] = sso_db_secret_name
	
	f, err := rev.OpenOrchestration("sso.yaml")
	if err != nil {
		return err
	}

	s, err := renderOrchestration(f, templateContext)
	if err != nil {
		return err
	}

	m, err := mf.FromReader(strings.NewReader(s), c)
	if err != nil {
		return err
	}

	transforms := []mf.Transformer{
		mf.InjectNamespace(k.GetNamespace()),
	}

	err = m.Transform(transforms...)
	if err != nil {
		return err
	}

	_ = m.DeleteAll()
	
	return nil
}

func getSsoStatus(k *kabanerov1alpha2.Kabanero, c client.Client, reqLogger logr.Logger) (bool, error) {
	// If SSO is not enabled, then there is no status to report.
	if k.Spec.Sso.Enable == false {
		k.Status.Sso.Configured = sso_false
		k.Status.Sso.Ready = sso_false
		k.Status.Sso.Message = ""
		return true, nil
	}

	// Determine if the SSO components are available.
	k.Status.Sso.Configured = sso_true
	k.Status.Sso.Ready = sso_false
	k.Status.Sso.Message = ""

	ssoDeploymentConfigInstance := &appsv1.DeploymentConfig{}
	err := c.Get(context.Background(), types.NamespacedName{
		Name:      "sso",
		Namespace: k.ObjectMeta.Namespace}, ssoDeploymentConfigInstance)

	if err != nil {
		k.Status.Sso.Message = err.Error()
		return false, err
	}

	foundAvailableCondition := false
	for _, condition := range ssoDeploymentConfigInstance.Status.Conditions {
		if condition.Type == appsv1.DeploymentAvailable {
			if condition.Status != corev1.ConditionTrue {
				err = fmt.Errorf("The SSO DeploymentConfig reported that it is not available: %v", condition.Message)
				k.Status.Sso.Message = err.Error()
				return false, err
			}
			foundAvailableCondition = true
		}
	}

	if foundAvailableCondition == false {
		err = errors.New("The SSO DeploymentConfig did not contain an 'Available' condition")
		k.Status.Sso.Message = err.Error()
		return false, err
	}

	// Now check the postgresql DeploymentConfig in the same way.
	postgreDeploymentConfigInstance := &appsv1.DeploymentConfig{}
	err = c.Get(context.Background(), types.NamespacedName{
		Name:      "sso-postgresql",
		Namespace: k.ObjectMeta.Namespace}, postgreDeploymentConfigInstance)

	if err != nil {
		k.Status.Sso.Message = err.Error()
		return false, err
	}

	foundAvailableCondition = false
	for _, condition := range postgreDeploymentConfigInstance.Status.Conditions {
		if condition.Type == appsv1.DeploymentAvailable {
			if condition.Status != corev1.ConditionTrue {
				err = fmt.Errorf("The SSO-Postgre DeploymentConfig reported that it is not available: %v", condition.Message)
				k.Status.Sso.Message = err.Error()
				return false, err
			}
			foundAvailableCondition = true
		}
	}

	if foundAvailableCondition == false {
		err = errors.New("The SSO-Postgre DeploymentConfig did not contain an 'Available' condition")
		k.Status.Sso.Message = err.Error()
		return false, err
	}

	k.Status.Sso.Ready = sso_true
	return true, nil
}

// Creates the secret containing DB_USERNAME, DB_PASSWORD, JGROUPS_CLUSTER_PASSWORD
func createDbSecret(k *kabanerov1alpha2.Kabanero, c client.Client, reqLogger logr.Logger) error {

	// Check if the Secret already exists.
	secretInstance := &corev1.Secret{}
	err := c.Get(context.Background(), types.NamespacedName{
		Name:      sso_db_secret_name,
		Namespace: k.ObjectMeta.Namespace}, secretInstance)

	if err != nil {
		if kerrors.IsNotFound(err) == false {
			return err
		}

		// Not found.  Make a new one.
		var ownerRef metav1.OwnerReference
		ownerRef, err = getOwnerReference(k, c, reqLogger)
		if err != nil {
			return err
		}

		secretInstance := &corev1.Secret{}
		secretInstance.ObjectMeta.Name = sso_db_secret_name
		secretInstance.ObjectMeta.Namespace = k.ObjectMeta.Namespace
		secretInstance.ObjectMeta.OwnerReferences = append(secretInstance.ObjectMeta.OwnerReferences, ownerRef)

		secretMap := make(map[string]string)
		secretMap["DB_USERNAME"] = randSecret(16)
		secretMap["DB_PASSWORD"] = randSecret(32)
		secretMap["JGROUPS_CLUSTER_PASSWORD"] = randSecret(32)
		
		secretInstance.StringData = secretMap

		reqLogger.Info(fmt.Sprintf("Attempting to create the SSO DB secret"))
		err = c.Create(context.TODO(), secretInstance)
	}

	return err
}


// Generate a random username, password
// Rules: Minimum Length: 9, 2 Digits, 2 Uppers, 2 Lowers
// Specials may break sed in sso startup
func randSecret(length int) string {

	if length < 9 {
		length = 9
	}

	rand.Seed(time.Now().UnixNano())
	digits := "0123456789"
	lowers := "abcdefghijklmnopqrstuvwxyz"
	uppers := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	all := digits + lowers + uppers

	buf := make([]byte, length)
	buf[0] = digits[rand.Intn(len(digits))]
	buf[1] = digits[rand.Intn(len(digits))]
	buf[2] = lowers[rand.Intn(len(lowers))]
	buf[3] = lowers[rand.Intn(len(lowers))]
	buf[4] = uppers[rand.Intn(len(uppers))]
	buf[5] = uppers[rand.Intn(len(uppers))]
	for i := 6; i < length; i++ {
			buf[i] = all[rand.Intn(len(all))]
	}
	rand.Shuffle(len(buf), func(i, j int) {
			buf[i], buf[j] = buf[j], buf[i]
	})
	
	return string(buf)
}
