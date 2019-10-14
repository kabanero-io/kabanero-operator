package kabaneroplatform

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"github.com/go-logr/logr"
	kabanerov1alpha1 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha1"
	mf "github.com/kabanero-io/manifestival"
	routev1 "github.com/openshift/api/route/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"math/big"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

func reconcileWebhook(ctx context.Context, k *kabanerov1alpha1.Kabanero, cl client.Client, reqLogger logr.Logger) error {

	// The Webhook entry was not configured in the spec.  We should disable it.
	if k.Spec.Webhook.Enable == false {
		cleanupWebhook(ctx, k, cl)
		return nil
	}

	// Deploy the Kabanero webhook components - service acct, role, etc
	rev, err := resolveSoftwareRevision(k, "webhook", k.Spec.Webhook.Version)
	if err != nil {
		return err
	}

	//The context which will be used to render any templates
	templateContext := rev.Identifiers

	image, err := imageUriWithOverrides(k.Spec.Webhook.Repository, k.Spec.Webhook.Tag, k.Spec.Webhook.Image, rev)
	if err != nil {
		return err
	}
	templateContext["image"] = image

	f, err := rev.OpenOrchestration("kabanero-webhook.yaml")
	if err != nil {
		return err
	}

	s, err := renderOrchestration(f, templateContext)
	if err != nil {
		return err
	}

	m, err := mf.FromReader(strings.NewReader(s), cl)
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

	// Create the default webhook secret, if we don't already have one
	err = createDefaultWebhookSecret(k, cl, reqLogger)
	if err != nil {
		return err
	}

	return nil
}

// Remove the webhook resources
func cleanupWebhook(ctx context.Context, k *kabanerov1alpha1.Kabanero, cl client.Client) error {
	rev, err := resolveSoftwareRevision(k, "webhook", k.Spec.Webhook.Version)
	if err != nil {
		return err
	}

	templateCtx := rev.Identifiers
	image, err := imageUriWithOverrides(k.Spec.Webhook.Repository, k.Spec.Webhook.Tag, k.Spec.Webhook.Image, rev)
	if err != nil {
		return err
	}

	templateCtx["image"] = image

	f, err := rev.OpenOrchestration("kabanero-webhook.yaml")
	if err != nil {
		return err
	}

	s, err := renderOrchestration(f, templateCtx)
	if err != nil {
		return err
	}

	m, err := mf.FromReader(strings.NewReader(s), cl)
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

	err = m.DeleteAll()
	if err != nil {
		return err
	}

	// Delete the secret.
	// Check if the ConfigMap resource already exists.
	secret := &corev1.Secret{}
	err = cl.Get(context.Background(), types.NamespacedName{
		Name:      "default-webhook-secret",
		Namespace: k.ObjectMeta.Namespace}, secret)

	if err != nil {
		// If it's not a not-found error, report it.
		if errors.IsNotFound(err) == false {
			return err
		}
	} else {
		// Need to delete it.
		err = cl.Delete(context.TODO(), secret)
		if err != nil {
			return err
		}
	}

	return nil
}


// Tries to see if the webhook route has been assigned a hostname.
func getWebhookRouteStatus(k *kabanerov1alpha1.Kabanero, cl client.Client, reqLogger logr.Logger) (bool, error) {

	// If disabled. Nothing to do. No need to display status if disabled.
	if k.Spec.Webhook.Enable == false {
		k.Status.Webhook = nil
		return true, nil
	}

	k.Status.Webhook = &kabanerov1alpha1.WebhookStatus{}
	k.Status.Webhook.Ready = "False"
	
	// Check that the route is accepted
	webhookRoute := &routev1.Route{}
	webhookRouteName := types.NamespacedName{Namespace: k.ObjectMeta.Namespace, Name: "kabanero-webhook"}
	err := cl.Get(context.TODO(), webhookRouteName, webhookRoute)
	if err == nil {
		k.Status.Webhook.Hostnames = nil
		// Looking for an ingress that has an admitted status and a hostname
		for _, ingress := range webhookRoute.Status.Ingress {
			var routeAdmitted bool = false
			for _, condition := range ingress.Conditions {
				if condition.Type == routev1.RouteAdmitted && condition.Status == corev1.ConditionTrue {
					routeAdmitted = true
				}
			}
			if routeAdmitted == true && len(ingress.Host) > 0 {
				k.Status.Webhook.Hostnames = append(k.Status.Webhook.Hostnames, ingress.Host)
			}
		}
		// If we found a hostname from an admitted route, we're done.
		if len(k.Status.Webhook.Hostnames) > 0 {
			k.Status.Webhook.Ready = "True"
			k.Status.Webhook.ErrorMessage = ""
		} else {
			k.Status.Webhook.Ready = "False"
			k.Status.Webhook.ErrorMessage = "There were no accepted ingress objects in the Route"
			return false, err
		}
	} else {
		var message string
		if errors.IsNotFound(err) {
			message = "The Route object for the webhook was not found"
		} else {
			message = "An error occurred retrieving the Route object for the webhook"
		}
		reqLogger.Error(err, message)
		k.Status.Webhook.Ready = "False"
		k.Status.Webhook.ErrorMessage = message + ": " + err.Error()
		k.Status.Webhook.Hostnames = nil
		return false, err
	}

	return true, nil
}

// Creates the default webhook secret
func createDefaultWebhookSecret(k *kabanerov1alpha1.Kabanero, c client.Client, reqLogger logr.Logger) error {
	secretName := "default-webhook-secret"

	// Check if the Secret already exists.
	secretInstance := &corev1.Secret{}
	err := c.Get(context.Background(), types.NamespacedName{
		Name:      secretName,
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
		secretInstance.ObjectMeta.Name = secretName
		secretInstance.ObjectMeta.Namespace = k.ObjectMeta.Namespace
		secretInstance.ObjectMeta.OwnerReferences = append(secretInstance.ObjectMeta.OwnerReferences, ownerRef)

		// Generate a 16 character random value
		possibleChars := []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890!@#$%^&*()-=_+")
		maxVal := big.NewInt(int64(len(possibleChars)))
		var buf bytes.Buffer
		for i := 0; i < 16; i++ {
			curInt, randErr := rand.Int(rand.Reader, maxVal)
			if randErr != nil {
				return randErr
			}
			// Convert int to char
			buf.WriteByte(possibleChars[curInt.Int64()])
		}

		secretMap := make(map[string]string)
		secretMap["secret"] = buf.String()
		secretInstance.StringData = secretMap

		reqLogger.Info(fmt.Sprintf("Attempting to create the default webhook secret"))
		err = c.Create(context.TODO(), secretInstance)
	}

	return err
}
