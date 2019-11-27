package kabanero

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
		Name("mutating.kabanero.kabanero.io").
		Mutating().
		Operations(admissionregistrationv1beta1.Create, admissionregistrationv1beta1.Update).
		WithManager(*mgr).
		ForType(&kabanerov1alpha1.Kabanero{}).
		Handlers(&kabaneroMutator{}).
		Build()
}

// kabaneroMutator mutates kabaneros
type kabaneroMutator struct {
	client  client.Client
	decoder types.Decoder
}

// Implement admission.Handler so the controller can handle admission request.
// This no-op assignment ensures that the struct implements the interface.
var _ admission.Handler = &kabaneroMutator{}

// kabaneroMutator verifies that the kabanero version singleton and array
// are not in conflict.
func (a *kabaneroMutator) Handle(ctx context.Context, req types.Request) types.Response {
	kabanero := &kabanerov1alpha1.Kabanero{}

	err := a.decoder.Decode(req, kabanero)
	if err != nil {
		return admission.ErrorResponse(http.StatusBadRequest, err)
	}
	copy := kabanero.DeepCopy()

	err = a.mutatekabaneroFn(ctx, copy)
	if err != nil {
		return admission.ErrorResponse(http.StatusInternalServerError, err)
	}
	return admission.PatchResponse(kabanero, copy)
}

// mutatekabaneroFn add an annotation to the given pod
func (a *kabaneroMutator) mutatekabaneroFn(ctx context.Context, kabanero *kabanerov1alpha1.Kabanero) error {
	// TODO: Business logic
	return nil
}

// kabaneroMutator implements inject.Client.
// A client will be automatically injected.
var _ inject.Client = &kabaneroMutator{}

// InjectClient injects the client.
func (v *kabaneroMutator) InjectClient(c client.Client) error {
	v.client = c
	return nil
}

// podAnnotator implements inject.Decoder.
// A decoder will be automatically injected.
var _ inject.Decoder = &kabaneroMutator{}

// InjectDecoder injects the decoder.
func (v *kabaneroMutator) InjectDecoder(d types.Decoder) error {
	v.decoder = d
	return nil
}
