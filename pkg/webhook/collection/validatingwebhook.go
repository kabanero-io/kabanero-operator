package collection

// The controller-runtime example webhook (v0.10) was used to build this
// webhook implementation.

import (
	"context"
	"net/http"

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
		Name("validating.collection.kabanero.io").
		Validating().
		Operations(admissionregistrationv1beta1.Create, admissionregistrationv1beta1.Update).
		WithManager(*mgr).
		ForType(&kabanerov1alpha1.Collection{}).
		Handlers(&collectionValidator{}).
		Build()
}

// collectionValidator validates Collections
type collectionValidator struct {
	client  client.Client
	decoder types.Decoder
}

// Implement admission.Handler so the controller can handle admission request.
// This no-op assignment ensures that the struct implements the interface.
var _ admission.Handler = &collectionValidator{}

// collectionValidator admits a collection if it passes validity checks
func (v *collectionValidator) Handle(ctx context.Context, req types.Request) types.Response {
	collection := &kabanerov1alpha1.Collection{}

	err := v.decoder.Decode(req, collection)
	if err != nil {
		return admission.ErrorResponse(http.StatusBadRequest, err)
	}

	allowed, reason, err := v.validateCollectionFn(ctx, collection)
	if err != nil {
		return admission.ErrorResponse(http.StatusInternalServerError, err)
	}
	return admission.ValidationResponse(allowed, reason)
}

func (v *collectionValidator) validateCollectionFn(ctx context.Context, pod *kabanerov1alpha1.Collection) (bool, string, error) {
	// For now, just reject everything.
	return true, "All collections are approved", nil
}

// collectionValidator implements inject.Client.
// A client will be automatically injected.
var _ inject.Client = &collectionValidator{}

// InjectClient injects the client.
func (v *collectionValidator) InjectClient(c client.Client) error {
	v.client = c
	return nil
}

// podValidator implements inject.Decoder.
// A decoder will be automatically injected.
var _ inject.Decoder = &collectionValidator{}

// InjectDecoder injects the decoder.
func (v *collectionValidator) InjectDecoder(d types.Decoder) error {
	v.decoder = d
	return nil
}
