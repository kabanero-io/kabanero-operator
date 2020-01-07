package collection

// The controller-runtime example webhook (v0.10) was used to build this
// webhook implementation.

import (
	"context"
	"fmt"
	"net/http"

	kabanerov1alpha1 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha1"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// BuildValidatingWebhook builds the webhook for the manager to register
func BuildValidatingWebhook(mgr *manager.Manager) *admission.Webhook {
	return &admission.Webhook{Handler: &collectionValidator{}}
}

// collectionValidator validates Collections
type collectionValidator struct {
	client  client.Client
	decoder *admission.Decoder
}

// Implement admission.Handler so the controller can handle admission request.
// This no-op assignment ensures that the struct implements the interface.
var _ admission.Handler = &collectionValidator{}

// collectionValidator admits a collection if it passes validity checks
func (v *collectionValidator) Handle(ctx context.Context, req admission.Request) admission.Response {
	collection := &kabanerov1alpha1.Collection{}

	err := v.decoder.Decode(req, collection)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	allowed, reason, err := v.validateCollectionFn(ctx, collection)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}
	return admission.ValidationResponse(allowed, reason)
}

func (v *collectionValidator) validateCollectionFn(ctx context.Context, collection *kabanerov1alpha1.Collection) (bool, string, error) {
	allowed := collection.Spec.Version == collection.Spec.Versions[0].Version &&
		collection.Spec.RepositoryUrl == collection.Spec.Versions[0].RepositoryUrl &&
		collection.Spec.DesiredState == collection.Spec.Versions[0].DesiredState

	if !allowed {
		reason := fmt.Sprintf("Single version collection model values do not match multiple collection model values. collection: %v", collection)
		err := fmt.Errorf(reason)
		return false, reason, err
	}

	return true, "", nil
}

// InjectClient injects the client.
func (v *collectionValidator) InjectClient(c client.Client) error {
	v.client = c
	return nil
}

// InjectDecoder injects the decoder.
func (v *collectionValidator) InjectDecoder(d *admission.Decoder) error {
	v.decoder = d
	return nil
}
