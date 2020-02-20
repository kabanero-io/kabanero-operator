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

	mf "github.com/manifestival/manifestival"
	mfc "github.com/manifestival/controller-runtime-client"
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
	
	// Go make sure that the necessary secret has been created.
	err = checkSecret(ctx, k, c, reqLogger)
	if err != nil {
		return err
	}
	
	//The context which will be used to render any templates
	templateContext := make(map[string]interface{})
	templateContext["ssoAdminSecretName"] = k.Spec.Sso.AdminSecretName

	// OpenShift modifies the spec section of the deployment config after we've deployed it.
	// That means that manifestival will try and change it back when it runs.  To prevent
	// that, we're going to try and insert the fields that change, if they already exist.
	postgreDeploymentConfigInstance := &appsv1.DeploymentConfig{}
	err = c.Get(context.Background(), types.NamespacedName{
		Name:      "sso-postgresql",
		Namespace: k.ObjectMeta.Namespace}, postgreDeploymentConfigInstance)

	if (err != nil) || (len(postgreDeploymentConfigInstance.Spec.Template.Spec.Containers) != 1) || (len(postgreDeploymentConfigInstance.Spec.Template.Spec.Containers[0].Image) == 0) {
		templateContext["postgreImage"] = "postgresql"
	} else {
		templateContext["postgreImage"] = postgreDeploymentConfigInstance.Spec.Template.Spec.Containers[0].Image
		}

	ssoDeploymentConfigInstance := &appsv1.DeploymentConfig{}
	err = c.Get(context.Background(), types.NamespacedName{
		Name:      "sso",
		Namespace: k.ObjectMeta.Namespace}, ssoDeploymentConfigInstance)

	if (err != nil) || (len(ssoDeploymentConfigInstance.Spec.Template.Spec.Containers) != 1) || (len(ssoDeploymentConfigInstance.Spec.Template.Spec.Containers[0].Image) == 0) {
		templateContext["ssoImage"] = "sso"
	} else {
		templateContext["ssoImage"] = ssoDeploymentConfigInstance.Spec.Template.Spec.Containers[0].Image
	}

	// Create DB secret if it does not exist
	err = createDbSecret(k, c, reqLogger)
	if err != nil {
		return fmt.Errorf("Failed to create the SSO DB secret: %v", err.Error())
	}
	templateContext["ssoDbSecretName"] = sso_db_secret_name

	f, err := rev.OpenOrchestration("sso.yaml")
	if err != nil {
		return err
	}

	s, err := renderOrchestration(f, templateContext)
	if err != nil {
		return err
	}

	mOrig, err := mf.ManifestFrom(mf.Reader(strings.NewReader(s)), mf.UseClient(mfc.NewClient(c)), mf.UseLogger(reqLogger.WithName("manifestival")))
	if err != nil {
		return err
	}

	transforms := []mf.Transformer{
		mf.InjectOwner(k),
		mf.InjectNamespace(k.GetNamespace()),
	}

	m, err := mOrig.Transform(transforms...)
	if err != nil {
		return err
	}

	err = m.Apply()
	if err != nil {
		return err
	}
	
	return nil
}

// Checks to make sure the secret required by the SSO configuration has
// been created and contains the required keys.
func checkSecret(ctx context.Context, k *kabanerov1alpha2.Kabanero, c client.Client, reqLogger logr.Logger) error {

	if len(k.Spec.Sso.AdminSecretName) == 0 {
		return errors.New("The SSO admin secret name must be specified in the Kabanero CR instance")
	}
	
	secretInstance := &corev1.Secret{}
	err := c.Get(context.Background(), types.NamespacedName{
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
	// too much.
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

	mOrig, err := mf.ManifestFrom(mf.Reader(strings.NewReader(s)), mf.UseClient(mfc.NewClient(c)), mf.UseLogger(reqLogger.WithName("manifestival")))
	if err != nil {
		return err
	}

	transforms := []mf.Transformer{
		mf.InjectNamespace(k.GetNamespace()),
	}

	m, err := mOrig.Transform(transforms...)
	if err != nil {
		return err
	}

	_ = m.Delete()
	
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

	// Make sure the configuration is correct
	k.Status.Sso.Configured = sso_true
	k.Status.Sso.Ready = sso_false
	k.Status.Sso.Message = ""

	err := checkSecret(context.Background(), k, c, reqLogger)
	if err != nil {
		k.Status.Sso.Message = err.Error()
		return false, err
	}

	// Before checking the deployment configs, check specifically if the
	// postgresql pod is waiting for a persistent volume (PV).
	podList := &corev1.PodList{}
	err = c.List(context.Background(), podList, client.InNamespace(k.GetNamespace()), client.MatchingLabels{"application": "sso", "deploymentConfig": "sso-postgresql"})
	if err == nil {
		for _, pod := range podList.Items {
			if pod.Status.Phase == corev1.PodPending {
				for _, condition := range pod.Status.Conditions {
					if (condition.Type == corev1.PodScheduled) && (condition.Status == corev1.ConditionFalse) && (condition.Reason == corev1.PodReasonUnschedulable) {
						// There is a reason the pod cannot be scheduled.  Lets tell the user what it is.
						err = fmt.Errorf("SSO-postgre pod %v cannot be scheduled: %v", pod.Name, condition.Message)
						k.Status.Sso.Message = err.Error()
						return false, err
					}
				}
			}
		}
	}
	
	// Determine if the postgressl SSO components are available.
	postgreDeploymentConfigInstance := &appsv1.DeploymentConfig{}
	err = c.Get(context.Background(), types.NamespacedName{
		Name:      "sso-postgresql",
		Namespace: k.ObjectMeta.Namespace}, postgreDeploymentConfigInstance)

	if err != nil {
		k.Status.Sso.Message = err.Error()
		return false, err
	}

	foundAvailableCondition := false
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

	// Check if the SSO components are available
	ssoDeploymentConfigInstance := &appsv1.DeploymentConfig{}
	err = c.Get(context.Background(), types.NamespacedName{
		Name:      "sso",
		Namespace: k.ObjectMeta.Namespace}, ssoDeploymentConfigInstance)

	if err != nil {
		k.Status.Sso.Message = err.Error()
		return false, err
	}

	foundAvailableCondition = false
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
