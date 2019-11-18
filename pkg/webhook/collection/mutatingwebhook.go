package collection

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
func BuildMutatingWebhook(mgr *manager.Manager) (webhook.Webhook, error) {
	// Create the mutating webhook
	return builder.NewWebhookBuilder().
		Name("mutating.collection.kabanero.io").
		Mutating().
		Operations(admissionregistrationv1beta1.Create, admissionregistrationv1beta1.Update).
		WithManager(*mgr).
		ForType(&kabanerov1alpha1.Collection{}).
		Handlers(&collectionMutator{}).
		Build()
}

// collectionMutator mutates collections
type collectionMutator struct {
	client  client.Client
	decoder types.Decoder
}

// Implement admission.Handler so the controller can handle admission request.
// This no-op assignment ensures that the struct implements the interface.
var _ admission.Handler = &collectionMutator{}

// collectionMutator verifies that the collection version singleton and array
// are not in conflict.
func (a *collectionMutator) Handle(ctx context.Context, req types.Request) types.Response {
	collection := &kabanerov1alpha1.Collection{}

	err := a.decoder.Decode(req, collection)
	if err != nil {
		return admission.ErrorResponse(http.StatusBadRequest, err)
	}
	copy := collection.DeepCopy()

	err = a.mutateCollectionFn(ctx, copy)
	if err != nil {
		return admission.ErrorResponse(http.StatusInternalServerError, err)
	}
	return admission.PatchResponse(collection, copy)
}

// mutateCollectionFn add an annotation to the given pod
func (a *collectionMutator) mutateCollectionFn(ctx context.Context, collection *kabanerov1alpha1.Collection) error {
	// TODO: Business logic
	return nil
}

// collectionMutator implements inject.Client.
// A client will be automatically injected.
var _ inject.Client = &collectionMutator{}

// InjectClient injects the client.
func (v *collectionMutator) InjectClient(c client.Client) error {
	v.client = c
	return nil
}

// podAnnotator implements inject.Decoder.
// A decoder will be automatically injected.
var _ inject.Decoder = &collectionMutator{}

// InjectDecoder injects the decoder.
func (v *collectionMutator) InjectDecoder(d types.Decoder) error {
	v.decoder = d
	return nil
}
