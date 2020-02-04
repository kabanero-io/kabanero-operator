package stack

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	kabanerov1alpha2 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha2"
	"k8s.io/apimachinery/pkg/api/errors"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// BuildMutatingWebhook builds the webhook for the manager to register
func BuildMutatingWebhook(mgr *manager.Manager) *admission.Webhook {
	return &admission.Webhook{Handler: &stackMutator{}}
}

// stackMutator mutates stacks
type stackMutator struct {
	client  client.Client
	decoder *admission.Decoder
}

// Implement admission.Handler so the controller can handle admission request.
// This no-op assignment ensures that the struct implements the interface.
var _ admission.Handler = &stackMutator{}

// stackMutator verifies that the stack version singleton and array
// are not in conflict.
func (a *stackMutator) Handle(ctx context.Context, req admission.Request) admission.Response {
	stack := &kabanerov1alpha2.Stack{}

	err := a.decoder.Decode(req, stack)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	err = a.mutateStackFn(ctx, stack)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	marshaledStack, err := json.Marshal(stack)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}
	
	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledStack)
}

// mutateStackFn updates stack version entries.
func (a *stackMutator) mutateStackFn(ctx context.Context, stack *kabanerov1alpha2.Stack) error {
	// Get the currently installed stack.
	current := &kabanerov1alpha2.Stack{}
	err := a.client.Get(ctx, client.ObjectKey{
		Name:      stack.Name,
		Namespace: stack.Namespace}, current)
	if err != nil {
		if !errors.IsNotFound(err) {
			return fmt.Errorf("Unable to retrieve installed stack object. Error: %v", err)
		}
	}

	err = processUpdate(current, stack)

	return err
}

// No update mutations are needed for Stacks at this time
func processUpdate(current *kabanerov1alpha2.Stack, new *kabanerov1alpha2.Stack) error {

	return nil
}



// InjectClient injects the client.
func (v *stackMutator) InjectClient(c client.Client) error {
	v.client = c
	return nil
}

// InjectDecoder injects the decoder.
func (v *stackMutator) InjectDecoder(d *admission.Decoder) error {
	v.decoder = d
	return nil
}
