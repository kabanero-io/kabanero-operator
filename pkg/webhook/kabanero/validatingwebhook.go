package kabanero

// The controller-runtime example webhook (v0.10) was used to build this
// webhook implementation.

import (
	"context"
	"net/http"
	"fmt"

	kabanerov1alpha1 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha1"

	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission/builder"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission/types"
)

// Builds the webhook for the manager to register
func BuildValidatingWebhook(mgr *manager.Manager) (webhook.Webhook, error) {
	// Create the validating webhook
	return builder.NewWebhookBuilder().
		Name("validating.kabanero.kabanero.io").
		Validating().
		Operations(admissionregistrationv1beta1.Create, admissionregistrationv1beta1.Update).
		WithManager(*mgr).
		ForType(&kabanerov1alpha1.Kabanero{}).
		Handlers(&kabaneroValidator{}).
		Build()
}

// kabaneroValidator validates kabaneros
type kabaneroValidator struct {
	client  client.Client
	decoder types.Decoder
}

// Implement admission.Handler so the controller can handle admission request.
// This no-op assignment ensures that the struct implements the interface.
var _ admission.Handler = &kabaneroValidator{}

// kabaneroValidator admits a kabanero if it passes validity checks
func (v *kabaneroValidator) Handle(ctx context.Context, req types.Request) types.Response {
	kabanero := &kabanerov1alpha1.Kabanero{}

	err := v.decoder.Decode(req, kabanero)
	if err != nil {
		return admission.ErrorResponse(http.StatusBadRequest, err)
	}

	allowed, reason, err := v.validatekabaneroFn(ctx, kabanero)
	if err != nil {
		return admission.ErrorResponse(http.StatusInternalServerError, err)
	}
	return admission.ValidationResponse(allowed, reason)
}

func (v *kabaneroValidator) validatekabaneroFn(ctx context.Context, pod *kabanerov1alpha1.Kabanero) (bool, string, error) {
	// For now, just reject everything.

	name := pod.ObjectMeta.Name
	namespace := pod.ObjectMeta.Namespace
	kabaneroList := &kabanerov1alpha1.KabaneroList{}
	options := &client.ListOptions{Namespace: namespace}

	err := v.client.List(ctx, options, kabaneroList)
	if err != nil {
		return false, fmt.Sprintf("Failed to list Kabaneros in namespace: %s", namespace), err
	}
	
	// If there is a Kabanero in this namespace other than a named match (update), reject. 
	// Do not allow more than 1 Kabanero per namespace.
	allow := true
	for _, kabanero := range kabaneroList.Items {
		if name == kabanero.Name {
			// Matching name, allow Update
			break
		} else {
			// This is an additional instance, reject
			allow = false
		}
	}
	
	if allow {
		return true, fmt.Sprintf("Kabanero %s in namespace %s approved", name, namespace), nil
	} else {
		return false, fmt.Sprintf("Rejecting additional Kabanero instance: %s in namespace: %s. Multiple Kabanero instances are not allowed.", name, namespace), nil
	}
}

// kabaneroValidator implements inject.Client.
// A client will be automatically injected.
var _ inject.Client = &kabaneroValidator{}

// InjectClient injects the client.
func (v *kabaneroValidator) InjectClient(c client.Client) error {
	v.client = c
	return nil
}

// podValidator implements inject.Decoder.
// A decoder will be automatically injected.
var _ inject.Decoder = &kabaneroValidator{}

// InjectDecoder injects the decoder.
func (v *kabaneroValidator) InjectDecoder(d types.Decoder) error {
	v.decoder = d
	return nil
}
