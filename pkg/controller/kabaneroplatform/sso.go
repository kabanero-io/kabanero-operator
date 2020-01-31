package kabaneroplatform

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	kabanerov1alpha2 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha2"

	mf "github.com/kabanero-io/manifestival"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
	// TODO: Implement status
	return true, nil
}

