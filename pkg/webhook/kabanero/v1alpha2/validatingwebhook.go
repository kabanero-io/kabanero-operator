package kabanero

// The controller-runtime example webhook (v0.10) was used to build this
// webhook implementation.

import (
	"context"
	"fmt"
	"net/http"

	kabanerov1alpha2 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha2"

	kutils "github.com/kabanero-io/kabanero-operator/pkg/controller/kabaneroplatform/utils"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// Builds the webhook for the manager to register
func BuildValidatingWebhook(mgr *manager.Manager) *admission.Webhook {
	return &admission.Webhook{Handler: &kabaneroValidator{}}
}

// kabaneroValidator validates kabaneros
type kabaneroValidator struct {
	client  client.Client
	decoder *admission.Decoder
}

// Implement admission.Handler so the controller can handle admission request.
// This no-op assignment ensures that the struct implements the interface.
var _ admission.Handler = &kabaneroValidator{}

// kabaneroValidator admits a kabanero if it passes validity checks
func (v *kabaneroValidator) Handle(ctx context.Context, req admission.Request) admission.Response {
	kabanero := &kabanerov1alpha2.Kabanero{}
	err := v.decoder.Decode(req, kabanero)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	allowed, reason, err := v.validatekabaneroFn(ctx, kabanero)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	return admission.ValidationResponse(allowed, reason)
}

func (v *kabaneroValidator) validatekabaneroFn(ctx context.Context, kab *kabanerov1alpha2.Kabanero) (bool, string, error) {
	allowed, reason, err := isKabaneroInstanceAllowed(v.client, ctx, kab)
	if !allowed {
		return allowed, reason, err
	}

	allowed, reason, err = kutils.ValidateGovernanceStackPolicy(kab)
	if !allowed {
		return allowed, reason, err
	}

	return true, "", nil
}

// InjectClient injects the client.
func (v *kabaneroValidator) InjectClient(c client.Client) error {
	v.client = c
	return nil
}

// InjectDecoder injects the decoder.
func (v *kabaneroValidator) InjectDecoder(d *admission.Decoder) error {
	v.decoder = d
	return nil
}

// Validates that no more than one kabanero instance in a given namespace is allowed.
func isKabaneroInstanceAllowed(cl client.Client, ctx context.Context, kab *kabanerov1alpha2.Kabanero) (bool, string, error) {
	name := kab.ObjectMeta.Name
	namespace := kab.ObjectMeta.Namespace
	kabaneroList := &kabanerov1alpha2.KabaneroList{}
	options := []client.ListOption{client.InNamespace(namespace)}
	err := cl.List(ctx, kabaneroList, options...)
	if err != nil {
		return false, fmt.Sprintf("Failed to list Kabaneros in namespace: %s", namespace), err
	}

	for _, kabanero := range kabaneroList.Items {
		if name == kabanero.Name {
			// Matching name, allow Update
			break
		} else {
			// This is an additional instance. Reject it.
			return false, fmt.Sprintf("Rejecting additional Kabanero instance: %s in namespace: %s. Multiple Kabanero instances are not allowed.", name, namespace), nil
		}
	}

	return true, "", nil
}
