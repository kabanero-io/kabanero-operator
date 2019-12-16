package collection

import (
	"context"
	"fmt"
	"net/http"

	kabanerov1alpha1 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha1"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission/builder"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission/types"
)

// BuildMutatingWebhook builds the webhook for the manager to register
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
	decoder types.Decoder // TODO should this be admission decoder?
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

// mutateCollectionFn updates collection version entries.
func (a *collectionMutator) mutateCollectionFn(ctx context.Context, collection *kabanerov1alpha1.Collection) error {
	// Get the currently installed collection.
	current := &kabanerov1alpha1.Collection{}
	err := a.client.Get(ctx, client.ObjectKey{
		Name:      collection.Name,
		Namespace: collection.Namespace}, current)
	if err != nil {
		if !errors.IsNotFound(err) {
			return fmt.Errorf("Unable to retrieve installed collection object. Error: %v", err)
		}
	}

	err = processUpdate(current, collection)

	return err
}

// Processes the needed changes when a collection update takes place. More precisely, it makes sure that
// that the values in new collection.Spec.Version[0] match the ones in new collection.Spec prior to processing the udpate.
func processUpdate(current *kabanerov1alpha1.Collection, new *kabanerov1alpha1.Collection) error {
	// New collection.Spec.versions[0] is defined.
	if len(new.Spec.Versions) > 0 {
		// New collection.Spec and new collection.Spec.Versions[0] values are the same.
		if areMixedVersionModelCollectionsEqual(new, new) {
			// Make a basic check to see if new collection.Spec.Versions[0] and new colleciton.Spec values are the same because they are cleared.
			if areSpecVersionValuesCleared(new) && areSpecVersions0ValuesCleared(new) {
				return fmt.Errorf("The new collection.Spec and collection.Spec.versions[0] were not specified. New Collection: %v. Current collection: %v", new, current)
			}

			return nil
		}

		// New colleciton.Spec != new collection.Spec.Versions[0].
		// Current colleciton.Spec == new collection.Spec.
		if areSpecVersionValuesEqual(current, new) {
			// Current collection.Spec != New collection.Spec.Versions[0].
			if !areMixedVersionModelCollectionsEqual(current, new) {
				// New Collection.Spec.Versions[0] values were cleared. Copy collection.Spec to collection.Spec.Versions[0].
				if areSpecVersions0ValuesCleared(new) {
					new.Spec.Versions[0].Version = new.Spec.Version
					new.Spec.Versions[0].RepositoryUrl = new.Spec.RepositoryUrl
					new.Spec.Versions[0].DesiredState = new.Spec.DesiredState
					return nil
				}

				// Update new collection.Spec with values with collection.Spec.Versions[0] values.
				new.Spec.Version = new.Spec.Versions[0].Version
				new.Spec.RepositoryUrl = new.Spec.Versions[0].RepositoryUrl
				new.Spec.DesiredState = new.Spec.Versions[0].DesiredState
				return nil
			}

			// No updates.
			return nil
		}

		// New colleciton.Spec != new collection.Spec.Versions[0].
		// Current colleciton.Spec != new collection.Spec.
		// Current collection.Spec == new collection.Spec.Versions[0].
		if areMixedVersionModelCollectionsEqual(current, new) {
			// New Collection.Spec values were cleared. Copy collection.Spec.Versions[0] to collection.Spec
			if areSpecVersionValuesCleared(new) {
				new.Spec.Version = new.Spec.Versions[0].Version
				new.Spec.RepositoryUrl = new.Spec.Versions[0].RepositoryUrl
				new.Spec.DesiredState = new.Spec.Versions[0].DesiredState
				return nil
			}
			// Update new collection.Spec.Versions[0] with new collection.Spec values.
			new.Spec.Versions[0].Version = new.Spec.Version
			new.Spec.Versions[0].RepositoryUrl = new.Spec.RepositoryUrl
			new.Spec.Versions[0].DesiredState = new.Spec.DesiredState
			return nil
		}

		// New colleciton.Spec != new collection.Spec.Versions[0].
		// Current colleciton.Spec != new collection.Spec.
		// Current collection.Spec != new collection.Spec.Versions[0].
		// Current collection.Spec.Versions[0] = new collection.Spec.Versions[0].
		if len(current.Spec.Versions) > 0 && areSpecVersions0ValuesEqual(current, new) {
			// Update new collection.Spec.Versions[0] with new collection.Spec values
			new.Spec.Versions[0].Version = new.Spec.Version
			new.Spec.Versions[0].RepositoryUrl = new.Spec.RepositoryUrl
			new.Spec.Versions[0].DesiredState = new.Spec.DesiredState
			return nil
		}

		return fmt.Errorf("current collection.Spec, current, collection.Spec.Versions[0], new collection.Spec, and new collection.Spec.versions[0] have different values. Invalid update. New Collection: %v. Current collection: %v", new, current)
	}

	// New collection.Spec.Versions[0] was NOT defined.
	// Collection.Spec values were cleared.
	if areSpecVersionValuesCleared(new) {
		return fmt.Errorf("The new collection information under Spec and Spec.versions[0] were. New Collection: %v. Current collection: %v", new, current)
	}

	// New collection.Spec.Versions[0] was NOT defined.
	// Collection.Spec values were NOT cleared.
	// Copy new collection.Spec to collection.Spec.Versions[0].
	versionsEntry := kabanerov1alpha1.CollectionVersion{
		Version:       new.Spec.Version,
		RepositoryUrl: new.Spec.RepositoryUrl,
		DesiredState:  new.Spec.DesiredState,
	}
	new.Spec.Versions = append(new.Spec.Versions, versionsEntry)
	if len(new.Spec.Versions) != 1 {
		return fmt.Errorf("Updated Spec.versions[] length of %v was not expected. Expected length: 1. New collection: %v", len(new.Spec.Versions), new)
	}

	return nil
}

// Returns true if all version related values in collection.Spec have a length of zero. Returns false otherwise.
func areSpecVersionValuesCleared(collection *kabanerov1alpha1.Collection) bool {
	return (len(collection.Spec.Version) == 0 &&
		len(collection.Spec.RepositoryUrl) == 0 &&
		len(collection.Spec.DesiredState) == 0)
}

// Returns true if all values in collection.Spec.Versions[0] have a length of zero. Returns false otherwise.
func areSpecVersions0ValuesCleared(collection *kabanerov1alpha1.Collection) bool {
	return (len(collection.Spec.Versions[0].Version) == 0 &&
		len(collection.Spec.Versions[0].RepositoryUrl) == 0 &&
		len(collection.Spec.Versions[0].DesiredState) == 0)
}

// Returns true if the version related values in the two input collection entries are the same. Returns false otherwise.
func areSpecVersionValuesEqual(this *kabanerov1alpha1.Collection, that *kabanerov1alpha1.Collection) bool {
	return this.Spec.Version == that.Spec.Version &&
		this.Spec.RepositoryUrl == that.Spec.RepositoryUrl &&
		this.Spec.DesiredState == that.Spec.DesiredState
}

// Returns true if collection.Spec.versions[0] values in the two input collection entries are the same. Returns false otherwise.
func areSpecVersions0ValuesEqual(this *kabanerov1alpha1.Collection, that *kabanerov1alpha1.Collection) bool {
	return this.Spec.Versions[0].Version == that.Spec.Versions[0].Version &&
		this.Spec.Versions[0].RepositoryUrl == that.Spec.Versions[0].RepositoryUrl &&
		this.Spec.Versions[0].DesiredState == that.Spec.Versions[0].DesiredState
}

// Returns true if version related values in input collection.Spec and input collection.Spec.Versions[0] are the same. Returns false otherwise.
func areMixedVersionModelCollectionsEqual(this *kabanerov1alpha1.Collection, that *kabanerov1alpha1.Collection) bool {
	return this.Spec.Version == that.Spec.Versions[0].Version &&
		this.Spec.RepositoryUrl == that.Spec.Versions[0].RepositoryUrl &&
		this.Spec.DesiredState == that.Spec.Versions[0].DesiredState
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
