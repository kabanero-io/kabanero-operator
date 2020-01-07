package kabanero

import (
	"context"
	"encoding/json"
	"net/http"

	kabanerov1alpha1 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha1"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// Builds the webhook for the manager to register
func BuildMutatingWebhook(mgr *manager.Manager) *admission.Webhook {
	return &admission.Webhook{Handler: &kabaneroMutator{}}
}

// kabaneroMutator mutates kabaneros
type kabaneroMutator struct {
	client  client.Client
	decoder *admission.Decoder
}

// Implement admission.Handler so the controller can handle admission request.
// This no-op assignment ensures that the struct implements the interface.
var _ admission.Handler = &kabaneroMutator{}

// kabaneroMutator verifies that the kabanero version singleton and array
// are not in conflict.
func (a *kabaneroMutator) Handle(ctx context.Context, req admission.Request) admission.Response {
	kabanero := &kabanerov1alpha1.Kabanero{}

	err := a.decoder.Decode(req, kabanero)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	err = a.mutatekabaneroFn(ctx, kabanero)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	marshaledKabanero, err := json.Marshal(kabanero)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledKabanero)
}

// mutatekabaneroFn add an annotation to the given pod
func (a *kabaneroMutator) mutatekabaneroFn(ctx context.Context, kabanero *kabanerov1alpha1.Kabanero) error {
	// TODO: Business logic
	return nil
}

// InjectClient injects the client.
func (v *kabaneroMutator) InjectClient(c client.Client) error {
	v.client = c
	return nil
}

// InjectDecoder injects the decoder.
func (v *kabaneroMutator) InjectDecoder(d *admission.Decoder) error {
	v.decoder = d
	return nil
}
