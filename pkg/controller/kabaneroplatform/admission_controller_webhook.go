package kabaneroplatform

import (
	"context"
	"encoding/base64"
	"fmt"
	"github.com/go-logr/logr"
	kabanerov1alpha2 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha2"

	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	
	mf "github.com/kabanero-io/manifestival"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

func reconcileAdmissionControllerWebhook(ctx context.Context, k *kabanerov1alpha2.Kabanero, c client.Client, reqLogger logr.Logger) error {

	// Figure out what version of the orchestration we are going to use.
	rev, err := resolveSoftwareRevision(k, "admission-webhook", k.Spec.AdmissionControllerWebhook.Version)
	if err != nil {
		return err
	}

	// For version 0.4.0 / orchestration v0.1:
	// We need to create a secret that the admission controller webhook will
	// populate with certificates.  This must be created outside of the
	// manifestival applies because we don't want to revert what the admission
	// controller webhook applies to it.
	if rev.Version == "0.4.0" {
		secretInstance := &corev1.Secret{}
		err := c.Get(context.Background(), types.NamespacedName{
			Name:      "kabanero-operator-admission-webhook",
			Namespace: k.ObjectMeta.Namespace}, secretInstance)

		if err != nil {
			if errors.IsNotFound(err) == false {
				return err
			}

			// Not found.  Make a new one.
			var ownerRef metav1.OwnerReference
			ownerRef, err = getOwnerReference(k, c, reqLogger)
			if err != nil {
				return err
			}

			secretInstance := &corev1.Secret{}
			secretInstance.ObjectMeta.Name = "kabanero-operator-admission-webhook"
			secretInstance.ObjectMeta.Namespace = k.ObjectMeta.Namespace
			secretInstance.ObjectMeta.OwnerReferences = append(secretInstance.ObjectMeta.OwnerReferences, ownerRef)
			
			reqLogger.Info("Attempting to create the admission controller webhook secret")
			err = c.Create(context.TODO(), secretInstance)
			
			if err != nil {
				return err
			}
		}
	}
	
	// Deploy the Kabanero admission controller webhook components - service acct, role, etc
	
	//The context which will be used to render any templates
	templateContext := rev.Identifiers

	image, err := imageUriWithOverrides(k.Spec.AdmissionControllerWebhook.Repository, k.Spec.AdmissionControllerWebhook.Tag, k.Spec.AdmissionControllerWebhook.Image, rev)
	if err != nil {
		return err
	}
	templateContext["image"] = image

	f, err := rev.OpenOrchestration("kabanero-operator-admission-webhook.yaml")
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

	// The webhook needs to use TLS.  Kabanero version 0.4.0 used controller-runtime
	// 0.1.x which generated its own certificates for webhooks.  Newer versions
	// of Kabanero use newer versions of controller-runtime which do not generate
	// their own certificates.  OpenShift is going to inject a certificate
	// into a secret, which the webhook pod will use.  The CA certificate needs
	// to be injected into the mutating webhook configuration and validating
	// webhook configuration, so that the Kube API server trusts the pod(s).
	if rev.Version != "0.4.0" {
		cmInstance := &corev1.ConfigMap{}
		err = c.Get(context.Background(), types.NamespacedName{
			Name:      "kabanero-operator-admission-webhook-ca-cert", 
			Namespace: k.GetNamespace()}, cmInstance)
		if err != nil {
			message := "The webhook configuration could not be created"
			reqLogger.Error(err, message)
			return fmt.Errorf("%v: %v", message, err.Error())
		}

		// See if the CA certificate was injected.
		caCert, ok := cmInstance.Data["service-ca.crt"]
		if !ok || caCert == "" {
			err = fmt.Errorf("The configmap did not have the service CA injected")
			reqLogger.Error(err, "Error creating webhook")
			return err
		}

		// Create the mutating webhook and validating webhook configuration
		encoded := base64.StdEncoding.EncodeToString([]byte(caCert))
		templateContext["caBundle"] = encoded
	
		f, err := rev.OpenOrchestration("kabanero-operator-admission-webhook-config.yaml")
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

		err = m.ApplyAll()
		if err != nil {
			return err
		}
	}
	
	return nil
}

// Removes the admission webhook server, as well as the resources
// created by controller-runtime that support the webhook.
func cleanupAdmissionControllerWebhook(k *kabanerov1alpha2.Kabanero, c client.Client) error {

	rev, err := resolveSoftwareRevision(k, "admission-webhook", k.Spec.AdmissionControllerWebhook.Version)
	if err != nil {
		return err
	}

	//The context which will be used to render any templates
	templateContext := rev.Identifiers

	image, err := imageUriWithOverrides(k.Spec.AdmissionControllerWebhook.Repository, k.Spec.AdmissionControllerWebhook.Tag, k.Spec.AdmissionControllerWebhook.Image, rev)
	if err != nil {
		return err
	}
	templateContext["image"] = image
  templateContext["caBundle"] = ""

	f, err := rev.OpenOrchestration("kabanero-operator-admission-webhook.yaml")
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

	transforms := []mf.Transformer{mf.InjectNamespace(k.GetNamespace())}
	err = m.Transform(transforms...)
	if err != nil {
		return err
	}

	err = m.DeleteAll()
	if err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
	}

	// The webhook configs are only created by manifestival later than Kabanero 0.4.0.
	if rev.Version != "0.4.0" {
		f, err := rev.OpenOrchestration("kabanero-operator-admission-webhook-config.yaml")
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
		
		err = m.DeleteAll()
		if err != nil {
			if !errors.IsNotFound(err) {
				return err
			}
		}
	}
	
	// Now, clean up the things that the controller-runtime created on
	// our behalf.
	secretInstance := &corev1.Secret{}
	secretInstance.Name = "kabanero-operator-admission-webhook"
	secretInstance.Namespace = k.GetNamespace()
	err = c.Delete(context.TODO(), secretInstance)

	if (err != nil) && (errors.IsNotFound(err) == false) {
		return err
	}

	// Some of these things are only created manually at version 0.4.0.
	if rev.Version == "0.4.0" {
		serviceInstance := &corev1.Service{}
		serviceInstance.Name = "kabanero-operator-admission-webhook"
		serviceInstance.Namespace = k.GetNamespace()
		err = c.Delete(context.TODO(), serviceInstance)

		if (err != nil) && (errors.IsNotFound(err) == false) {
			return err
		}

		mutatingWebhookConfigInstance := &admissionregistrationv1beta1.MutatingWebhookConfiguration{}
		mutatingWebhookConfigInstance.Name = "webhook.operator.kabanero.io"
		err = c.Delete(context.TODO(), mutatingWebhookConfigInstance)

		if (err != nil) && (errors.IsNotFound(err) == false) {
			return err
		}

		validatingWebhookConfigInstance := &admissionregistrationv1beta1.ValidatingWebhookConfiguration{}
		validatingWebhookConfigInstance.Name = "webhook.operator.kabanero.io"
		err = c.Delete(context.TODO(), validatingWebhookConfigInstance)

		if (err != nil) && (errors.IsNotFound(err) == false) {
			return err
		}
	}
		
	return nil
}

// Check to see if the admission controller webhook is set up correctly.

func getAdmissionControllerWebhookStatus(k *kabanerov1alpha2.Kabanero, c client.Client, reqLogger logr.Logger) (bool, error) {
	k.Status.AdmissionControllerWebhook.Ready = "False"
	k.Status.AdmissionControllerWebhook.ErrorMessage = ""

	// Check to see if the webhook pod has started and is available
	_, err := getDeploymentStatus(c, "kabanero-operator-admission-webhook", k.GetNamespace())
	if err != nil {
		message := "The admission webhook deployment was not ready: " + err.Error()
		reqLogger.Error(err, message)
		k.Status.AdmissionControllerWebhook.ErrorMessage = message
		return false, err
	}

	// Check to see if the mutating webhook was registered.
	mutatingWebhookConfigInstance := &admissionregistrationv1beta1.MutatingWebhookConfiguration{}
	err = c.Get(context.Background(), types.NamespacedName{
		Name:      "webhook.operator.kabanero.io", 
		Namespace: ""}, mutatingWebhookConfigInstance)

	if err != nil {
		message := "The admission webhook deployment was not ready: " + err.Error()
		reqLogger.Error(err, message)
		k.Status.AdmissionControllerWebhook.ErrorMessage = message
		return false, err
	}

	// Check to see if the validating webhook was registered.
	validatingWebhookConfigInstance := &admissionregistrationv1beta1.ValidatingWebhookConfiguration{}
	err = c.Get(context.Background(), types.NamespacedName{
		Name:      "webhook.operator.kabanero.io", 
		Namespace: ""}, validatingWebhookConfigInstance)

	if err != nil {
		message := "The admission webhook deployment was not ready: " + err.Error()
		reqLogger.Error(err, message)
		k.Status.AdmissionControllerWebhook.ErrorMessage = message
		return false, err
	}

	k.Status.AdmissionControllerWebhook.Ready = "True"
	return true, nil
}
	
