package kabaneroplatform

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	kabanerov1alpha2 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha2"

	mf "github.com/kabanero-io/manifestival"
	appsv1 "github.com/openshift/api/apps/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	sso_true string = "True"
	sso_false string = "False"
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
		reqLogger.Info(fmt.Sprintf("TDK SSO Pod list returned %v entries", len(podList.Items)))
		for _, pod := range podList.Items {
			reqLogger.Info(fmt.Sprintf("TDK pod name %v phase %v", pod.Name, pod.Status.Phase))
			if pod.Status.Phase == corev1.PodPending {
				reqLogger.Info(fmt.Sprintf("TDK condition list %v entries", len(pod.Status.Conditions)))
				for _, condition := range pod.Status.Conditions {
					reqLogger.Info(fmt.Sprintf("TDK condition type %v status %v reason %v", condition.Type, condition.Status, condition.Reason))
					if (condition.Type == corev1.PodScheduled) && (condition.Status == corev1.ConditionFalse) && (condition.Reason == corev1.PodReasonUnschedulable) {
						// There is a reason the pod cannot be scheduled.  Lets tell the user what it is.
						err = fmt.Errorf("SSO-postgre pod %v cannot be scheduled: %v", pod.Name, condition.Message)
						k.Status.Sso.Message = err.Error()
						return false, err
					}
				}
			}
		}
	} else {
		reqLogger.Error(err, fmt.Sprintf("TDK pod list error"))
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

