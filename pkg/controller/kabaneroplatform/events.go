package kabaneroplatform

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"github.com/go-logr/logr"
	kabanerov1alpha2 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha2"
	mf "github.com/manifestival/manifestival"
	mfc "github.com/manifestival/controller-runtime-client"
	routev1 "github.com/openshift/api/route/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"math/big"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

func reconcileEvents(ctx context.Context, k *kabanerov1alpha2.Kabanero, cl client.Client, reqLogger logr.Logger) error {

	// The Events entry was not configured in the spec.  We should disable it.
	if k.Spec.Events.Enable == false {
		cleanupEvents(ctx, k, cl, reqLogger)
		return nil
	}

	// Deploy the Kabanero events components - service acct, role, etc
	rev, err := resolveSoftwareRevision(k, "events", k.Spec.Events.Version)
	if err != nil {
		return err
	}

	//The context which will be used to render any templates
	templateContext := rev.Identifiers

	image, err := imageUriWithOverrides(k.Spec.Events.Repository, k.Spec.Events.Tag, k.Spec.Events.Image, rev)
	if err != nil {
		return err
	}
	templateContext["image"] = image
	templateContext["instance"] = k.ObjectMeta.UID

	f, err := rev.OpenOrchestration("kabanero-events.yaml")
	if err != nil {
		return err
	}

	s, err := renderOrchestration(f, templateContext)
	if err != nil {
		return err
	}

	mOrig, err := mf.ManifestFrom(mf.Reader(strings.NewReader(s)), mf.UseClient(mfc.NewClient(cl)), mf.UseLogger(reqLogger.WithName("manifestival")))
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

	// Create the default events secret, if we don't already have one
	err = createDefaultEventsSecret(k, cl, reqLogger)
	if err != nil {
		return err
	}

	return nil
}

// Remove the events resources
func cleanupEvents(ctx context.Context, k *kabanerov1alpha2.Kabanero, cl client.Client, reqLogger logr.Logger) error {
	rev, err := resolveSoftwareRevision(k, "events", k.Spec.Events.Version)
	if err != nil {
		return err
	}

	templateCtx := rev.Identifiers
	image, err := imageUriWithOverrides(k.Spec.Events.Repository, k.Spec.Events.Tag, k.Spec.Events.Image, rev)
	if err != nil {
		return err
	}

	templateCtx["image"] = image

	f, err := rev.OpenOrchestration("kabanero-events.yaml")
	if err != nil {
		return err
	}

	s, err := renderOrchestration(f, templateCtx)
	if err != nil {
		return err
	}

	mOrig, err := mf.ManifestFrom(mf.Reader(strings.NewReader(s)), mf.UseClient(mfc.NewClient(cl)), mf.UseLogger(reqLogger.WithName("manifestival")))
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

	err = m.Delete()
	if err != nil {
		return err
	}

	// Delete the secret.
	// Check if the ConfigMap resource already exists.
	secret := &corev1.Secret{}
	err = cl.Get(context.Background(), types.NamespacedName{
		Name:      "default-events-secret",
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

// Tries to see if the events route has been assigned a hostname.
func getEventsRouteStatus(k *kabanerov1alpha2.Kabanero, cl client.Client, reqLogger logr.Logger) (bool, error) {

	// If disabled. Nothing to do. No need to display status if disabled.
	if k.Spec.Events.Enable == false {
		k.Status.Events = nil
		return true, nil
	}

	k.Status.Events = &kabanerov1alpha2.EventsStatus{}
	k.Status.Events.Ready = "False"

	// Check that the route is accepted
	eventsRoute := &routev1.Route{}
	eventsRouteName := types.NamespacedName{Namespace: k.ObjectMeta.Namespace, Name: "kabanero-events"}
	err := cl.Get(context.TODO(), eventsRouteName, eventsRoute)
	if err == nil {
		k.Status.Events.Hostnames = nil
		// Looking for an ingress that has an admitted status and a hostname
		for _, ingress := range eventsRoute.Status.Ingress {
			var routeAdmitted bool = false
			for _, condition := range ingress.Conditions {
				if condition.Type == routev1.RouteAdmitted && condition.Status == corev1.ConditionTrue {
					routeAdmitted = true
				}
			}
			if routeAdmitted == true && len(ingress.Host) > 0 {
				k.Status.Events.Hostnames = append(k.Status.Events.Hostnames, ingress.Host)
			}
		}
		// If we found a hostname from an admitted route, we're done.
		if len(k.Status.Events.Hostnames) > 0 {
			k.Status.Events.Ready = "True"
			k.Status.Events.Message = ""
		} else {
			k.Status.Events.Ready = "False"
			k.Status.Events.Message = "There were no accepted ingress objects in the Route"
			return false, err
		}
	} else {
		var message string
		if errors.IsNotFound(err) {
			message = "The Route object for the events was not found"
		} else {
			message = "An error occurred retrieving the Route object for the events"
		}
		reqLogger.Error(err, message)
		k.Status.Events.Ready = "False"
		k.Status.Events.Message = message + ": " + err.Error()
		k.Status.Events.Hostnames = nil
		return false, err
	}

	return true, nil
}

// Creates the default events secret
func createDefaultEventsSecret(k *kabanerov1alpha2.Kabanero, c client.Client, reqLogger logr.Logger) error {
	secretName := "default-events-secret"

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

		reqLogger.Info(fmt.Sprintf("Attempting to create the default events secret"))
		err = c.Create(context.TODO(), secretInstance)
	}

	return err
}
