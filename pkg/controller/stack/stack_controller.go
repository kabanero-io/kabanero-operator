package stack

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"net/http"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	kabanerov1alpha2 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha2"
	sutils "github.com/kabanero-io/kabanero-operator/pkg/controller/stack/utils"
	"github.com/kabanero-io/kabanero-operator/pkg/controller/transforms"
	cutils "github.com/kabanero-io/kabanero-operator/pkg/controller/utils"
	mfc "github.com/manifestival/controller-runtime-client"
	mf "github.com/manifestival/manifestival"

	//	corev1 "k8s.io/api/core/v1"
	pipelinev1alpha1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	k8runtime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_stack")
var cIDRegex = regexp.MustCompile("^[a-z]([a-z0-9-]*[a-z0-9])?$")

const (
	// Asset status.
	assetStatusActive  = "active"
	assetStatusFailed  = "failed"
	assetStatusUnknown = "unknown"
)

// Add creates a new Stack Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileStack{client: mgr.GetClient(), scheme: mgr.GetScheme(), indexResolver: ResolveIndex}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("stack-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Create Stack predicate
	cPred := predicate.Funcs{
		GenericFunc: func(e event.GenericEvent) bool {
			return false
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			// Returning true only when the metadata generation has changed,
			// allows us to ignore events where only the object status has changed,
			// since the generation is not incremented when only the status changes
			return e.MetaOld.GetGeneration() != e.MetaNew.GetGeneration()
		},
	}

	// Watch for changes to primary resource Stack
	err = c.Watch(&source.Kind{Type: &kabanerov1alpha2.Stack{}}, &handler.EnqueueRequestForObject{}, cPred)
	if err != nil {
		return err
	}

	// Create a handler for handling Tekton Pipeline & Task events
	tH := &handler.EnqueueRequestForOwner{
		IsController: false,
		OwnerType:    &kabanerov1alpha2.Stack{},
	}

	// Create Tekton predicate
	tPred := predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			// ignore Create. Stack create applies the documents. Watch would unnecessarily requeue.
			return false
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return false
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			// Returning true only when the metadata generation has changed,
			// allows us to ignore events where only the object status has changed,
			// since the generation is not incremented when only the status changes
			return e.MetaOld.GetGeneration() != e.MetaNew.GetGeneration()
		},
	}

	// Watch for changes to Stack Tekton Pipeline objects
	err = c.Watch(&source.Kind{Type: &pipelinev1alpha1.Pipeline{}}, tH, tPred)
	if err != nil {
		log.Info(fmt.Sprintf("Tekton Pipelines may not be installed"))
		return err
	}

	err = c.Watch(&source.Kind{Type: &pipelinev1alpha1.Task{}}, tH, tPred)
	if err != nil {
		log.Info(fmt.Sprintf("Tekton Pipelines may not be installed"))
		return err
	}

	err = c.Watch(&source.Kind{Type: &pipelinev1alpha1.Condition{}}, tH, tPred)
	if err != nil {
		log.Info(fmt.Sprintf("Tekton Pipelines may not be installed"))
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileStack implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileStack{}

// ReconcileStack reconciles a Stack object
type ReconcileStack struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *k8runtime.Scheme

	//The indexResolver which will be used during reconciliation
	indexResolver func(client.Client, kabanerov1alpha2.RepositoryConfig, string, []Pipelines, []Trigger, string, logr.Logger) (*Index, error)
}

// Reconcile reads that state of the cluster for a Stack object and makes changes based on the state read
// and what is in the Stack.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileStack) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	ctx := context.Background()

	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling Stack")

	// Fetch the Stack instance
	instance := &kabanerov1alpha2.Stack{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// If the stack is being deleted, and our finalizer is set, process it.
	beingDeleted, err := processDeletion(ctx, instance, r.client, reqLogger)
	if err != nil {
		return reconcile.Result{}, err
	}

	if beingDeleted {
		return reconcile.Result{}, nil
	}

	rr, err := r.ReconcileStack(instance)

	r.client.Status().Update(ctx, instance)

	// Force a requeue if there are failed assets.  These should be retried, and since
	// they are hosted outside of Kubernetes, the controller will not see when they
	// are updated.
	if failedAssets(instance.Status) && (rr.Requeue == false) {
		reqLogger.Info("Forcing requeue due to failed assets in the Stack")
		rr.Requeue = true
		rr.RequeueAfter = 60 * time.Second
	}

	return rr, err
}

// Check to see if the status contains any assets that are failed
func failedAssets(status kabanerov1alpha2.StackStatus) bool {
	for _, version := range status.Versions {
		for _, pipeline := range version.Pipelines {
			for _, asset := range pipeline.ActiveAssets {
				if asset.Status == assetStatusFailed {
					return true
				}
			}
		}
	}
	return false
}

// Create a stack status summary string
func stackSummary(status kabanerov1alpha2.StackStatus) string {
	var summary = make([]string, len(status.Versions))
	for i, version := range status.Versions {
		summary[i] = fmt.Sprintf("%v: %v", version.Version, version.Status)
	}
	return fmt.Sprintf("[ %v ]", strings.Join(summary, ", "))
}

// Used internally by ReconcileStack to store matching stacks
// Could be less cumbersome to just use kabanerov1alpha2.Stack
type resolvedStack struct {
	repositoryURL string
	stack         Stack
}

// ReconcileStack activates or deactivates the input stack.
func (r *ReconcileStack) ReconcileStack(c *kabanerov1alpha2.Stack) (reconcile.Result, error) {
	r_log := log.WithValues("Request.Namespace", c.GetNamespace()).WithValues("Request.Name", c.GetName())

	// Clear the status message, we'll generate a new one if necessary
	c.Status.StatusMessage = ""

	//The stack name can be either the spec.name or the resource name. The
	//spec.name has precedence
	var stackName string
	if c.Spec.Name != "" {
		stackName = c.Spec.Name
	} else {
		stackName = c.Name
	}

	r_log = r_log.WithValues("Stack.Name", stackName)

	// Process the versions array and activate (or deactivate) the desired versions.
	err := reconcileActiveVersions(c, r.client, r_log)
	if err != nil {
		// TODO - what is useful to print?
		log.Error(err, fmt.Sprintf("Error during reconcileActiveVersions"))
	}

	return reconcile.Result{}, nil
}

// A key to the pipeline use count map.  Only one of "url" and "gitRelease" should
// be provided.  It's one or the other.
type pipelineUseMapKey struct {
	url        string
	gitRelease kabanerov1alpha2.GitReleaseInfo
	digest     string
}

// The value in the pipeline use count map
type pipelineUseMapValue struct {
	kabanerov1alpha2.PipelineStatus
	useCount      int64
	manifests     []StackAsset
	manifestError error
}

// A specific version of a pipeline zip in a specific version of a stack
type pipelineVersion struct {
	pipelineUseMapKey
	version string
}

// Some objects need to get created in a specific namespace.  Try and figure out what that is.
func getNamespaceForObject(u *unstructured.Unstructured, defaultNamespace string) string {
	kind := u.GetKind()

	// Presently, TriggerBinding, TriggerTemplate and EventListener objects are created
	// in the tekton-pipelines namespace.
	//
	// TODO (future): Kabanero Eventing will likely want to create these objects in the
	//                kabanero namespace, since they are not interacting with the Tekton
	//                Webhook Extension anymore.  We'll need to make this selectable
	//                somehow, perhaps just using the namespace in the yaml.
	if (kind == "TriggerBinding") || (kind == "TriggerTemplate") || (kind == "EventListener") {
		return "tekton-pipelines"
	}

	return defaultNamespace
}

func gitReleaseSpecToGitReleaseInfo(gitRelease kabanerov1alpha2.GitReleaseSpec) kabanerov1alpha2.GitReleaseInfo {
	return kabanerov1alpha2.GitReleaseInfo{Hostname: gitRelease.Hostname, Organization: gitRelease.Organization, Project: gitRelease.Project, Release: gitRelease.Release, AssetName: gitRelease.AssetName}
}

func reconcileActiveVersions(stackResource *kabanerov1alpha2.Stack, c client.Client, logger logr.Logger) error {

	// Gather the known stack asset (*-tasks, *-pipeline) substitution data.
	renderingContext := make(map[string]interface{})

	// The stack id is the name of the Appsody stack directory ("the stack name from the stack path").
	// Appsody stack creation namimg constrains the length to 68 characters:
	// "The name must start with a lowercase letter, contain only lowercase letters, numbers, or dashes,
	// and cannot end in a dash."
	cID := stackResource.Spec.Name
	if len(cID) > 68 {
		return fmt.Errorf("Failed to reconcile stack because an invalid stack id of %v was found. The stack id must must be 68 characters or less. For more details see the Appsody stack create command documentation", cID)
	}

	if !cIDRegex.MatchString(cID) {
		return fmt.Errorf("Failed to reconcile stack because an invalid stack id of %v was found. The stack id value must follow stack creation name rules. For more details see the Appsody stack create command documentation", cID)
	}

	renderingContext["CollectionId"] = cID
	renderingContext["StackId"] = cID

	ownerIsController := false
	assetOwner := metav1.OwnerReference{
		APIVersion: stackResource.TypeMeta.APIVersion,
		Kind:       stackResource.TypeMeta.Kind,
		Name:       stackResource.ObjectMeta.Name,
		UID:        stackResource.ObjectMeta.UID,
		Controller: &ownerIsController,
	}

	// Multiple versions of the same stack, could be using the same pipeline zip.  Count how many
	// times each pipeline has been used.
	assetUseMap := make(map[pipelineUseMapKey]*pipelineUseMapValue)
	for _, curStatus := range stackResource.Status.Versions {
		for _, pipeline := range curStatus.Pipelines {
			key := pipelineUseMapKey{digest: pipeline.Digest}
			if pipeline.GitRelease.IsUsable() {
				key.gitRelease = pipeline.GitRelease
			} else {
				key.url = pipeline.Url
			}
			value := assetUseMap[key]
			if value == nil {
				value = &pipelineUseMapValue{}
				pipeline.DeepCopyInto(&(value.PipelineStatus))
				assetUseMap[key] = value
			}
			value.useCount++
		}
	}

	// Reconcile the version changes.  Make a set of versions being removed, and versions being added.  Be
	// sure to take into consideration the digest on the individual pipeline zips.
	assetsToDecrement := make(map[pipelineVersion]bool)
	assetsToIncrement := make(map[pipelineVersion]bool)
	for _, curStatus := range stackResource.Status.Versions {
		for _, pipeline := range curStatus.Pipelines {
			key := pipelineUseMapKey{digest: pipeline.Digest}
			if pipeline.GitRelease.IsUsable() {
				key.gitRelease = pipeline.GitRelease
			} else {
				key.url = pipeline.Url
			}
			cur := pipelineVersion{pipelineUseMapKey: key, version: curStatus.Version}
			assetsToDecrement[cur] = true
		}
	}

	// When processing the pipelines currently referenced in the stack spec, save
	// off whether we should disable certificate verification checking per-resource.
	certVerification := make(map[pipelineUseMapKey]bool)
	for _, curSpec := range stackResource.Spec.Versions {
		if !strings.EqualFold(curSpec.DesiredState, kabanerov1alpha2.StackDesiredStateInactive) {
			for _, pipeline := range curSpec.Pipelines {
				key := pipelineUseMapKey{digest: pipeline.Sha256}
				if pipeline.GitRelease.IsUsable() {
					key.gitRelease = gitReleaseSpecToGitReleaseInfo(pipeline.GitRelease)
					certVerification[key] = pipeline.GitRelease.SkipCertVerification
				} else {
					key.url = pipeline.Https.Url
					certVerification[key] = pipeline.Https.SkipCertVerification
				}
				cur := pipelineVersion{pipelineUseMapKey: key, version: curSpec.Version}
				if assetsToDecrement[cur] == true {
					delete(assetsToDecrement, cur)
				} else {
					assetsToIncrement[cur] = true
				}
			}
		}
	}

	// Now go thru the maps and update the use counts
	for cur, _ := range assetsToDecrement {
		value := assetUseMap[cur.pipelineUseMapKey]
		if value == nil {
			return fmt.Errorf("Pipeline version not found in use map: %v", cur)
		}

		value.useCount--
	}

	for cur, _ := range assetsToIncrement {
		value := assetUseMap[cur.pipelineUseMapKey]
		if value == nil {
			// Need to add a new entry for this pipeline.
			value = &pipelineUseMapValue{PipelineStatus: kabanerov1alpha2.PipelineStatus{Url: cur.url, GitRelease: cur.gitRelease, Digest: cur.digest}}
			assetUseMap[cur.pipelineUseMapKey] = value
		}

		value.useCount++
	}

	// Now iterate thru the asset use map and delete any assets with a use count of 0,
	// and create any assets with a positive use count.
	for _, value := range assetUseMap {
		if value.useCount <= 0 {
			log.Info(fmt.Sprintf("Deleting assets with use count %v: %v", value.useCount, value))

			for _, asset := range value.ActiveAssets {
				// Old assets may not have a namespace set - correct that now.
				if len(asset.Namespace) == 0 {
					asset.Namespace = stackResource.GetNamespace()
				}

				deleteAsset(c, asset, assetOwner)
			}
		}
	}

	for key, value := range assetUseMap {
		if value.useCount > 0 {
			log.Info(fmt.Sprintf("Creating assets with use count %v: %v", value.useCount, value))

			// Check to see if there is already an asset list.  If not, read the manifests and
			// create one.
			if len(value.ActiveAssets) == 0 {
				// Add the Digest to the rendering context. No need to validate if the digest was tampered
				// with here. Later one and before we do anything with this, we will have validated the specified
				// digest against the generated digest from the archive.
				if len(value.Digest) >= 8 {
					renderingContext["Digest"] = value.Digest[0:8]
				} else {
					renderingContext["Digest"] = "nodigest"
				}

				// Retrieve manifests as unstructured.  If we could not get them, skip.
				manifests, err := GetManifests(c, stackResource.GetNamespace(), value.PipelineStatus, renderingContext, certVerification[key], log)
				if err != nil {
					log.Error(err, fmt.Sprintf("Error retrieving archive manifests: %v", value))
					value.manifestError = err
					continue
				}

				// Save the manifests for later.
				value.manifests = manifests

				// Create the asset status slice, but don't apply anything yet.
				for _, asset := range manifests {
					// Figure out what namespace we should create the object in.
					value.ActiveAssets = append(value.ActiveAssets, kabanerov1alpha2.RepositoryAssetStatus{
						Name:          asset.Name,
						Namespace:     getNamespaceForObject(&asset.Yaml, stackResource.GetNamespace()),
						Group:         asset.Group,
						Version:       asset.Version,
						Kind:          asset.Kind,
						Digest:        asset.Sha256,
						Status:        assetStatusUnknown,
						StatusMessage: "Asset has not been applied yet.",
					})
				}
			}

			// Now go thru the asset list and see if the objects are there.  If not, create them.
			for index, asset := range value.ActiveAssets {
				// Old assets may not have a namespace set - correct that now.
				if len(asset.Namespace) == 0 {
					asset.Namespace = stackResource.GetNamespace()
					value.ActiveAssets[index].Namespace = asset.Namespace
				}

				u := &unstructured.Unstructured{}
				u.SetGroupVersionKind(schema.GroupVersionKind{
					Group:   asset.Group,
					Version: asset.Version,
					Kind:    asset.Kind,
				})

				err := c.Get(context.Background(), client.ObjectKey{
					Namespace: asset.Namespace,
					Name:      asset.Name,
				}, u)

				if err != nil {
					if errors.IsNotFound(err) == false {
						log.Error(err, fmt.Sprintf("Unable to check asset name %v", asset.Name))
						value.ActiveAssets[index].Status = assetStatusUnknown
						value.ActiveAssets[index].StatusMessage = "Unable to check asset: " + err.Error()
					} else {
						// Make sure the manifests are loaded.
						if len(value.manifests) == 0 {
							// Add the Digest to the rendering context.
							if len(value.Digest) >= 8 {
								renderingContext["Digest"] = value.Digest[0:8]
							} else {
								renderingContext["Digest"] = "nodigest"
							}

							// Retrieve manifests as unstructured
							manifests, err := GetManifests(c, stackResource.GetNamespace(), value.PipelineStatus, renderingContext, certVerification[key], log)
							if err != nil {
								log.Error(err, fmt.Sprintf("Object %v not found and manifests not available: %v", asset.Name, value))
								value.ActiveAssets[index].Status = assetStatusFailed
								value.ActiveAssets[index].StatusMessage = "Manifests are no longer available at specified URL"
							} else {
								// Save the manifests for later.
								value.manifests = manifests
							}
						}

						// Now find the correct manifest and create the object
						for _, manifest := range value.manifests {
							if asset.Name == manifest.Name {
								resources := []unstructured.Unstructured{manifest.Yaml}

								// Only allow Group: tekton.dev
								allowed := true
								for _, resource := range resources {
									if resource.GroupVersionKind().Group != "tekton.dev" {
										value.ActiveAssets[index].Status = assetStatusFailed
										value.ActiveAssets[index].StatusMessage = "Manifest rejected: contains a Group not equal to tekton.dev"
										allowed = false
									}
								}

								if allowed == true {
									mOrig, err := mf.ManifestFrom(mf.Slice(resources), mf.UseClient(mfc.NewClient(c)), mf.UseLogger(log.WithName("manifestival")))

									log.Info(fmt.Sprintf("Resources: %v", mOrig.Resources()))

									transforms := []mf.Transformer{
										transforms.InjectOwnerReference(assetOwner),
										mf.InjectNamespace(asset.Namespace),
									}

									m, err := mOrig.Transform(transforms...)
									if err != nil {
										log.Error(err, fmt.Sprintf("Error transforming manifests for %v", asset.Name))
										value.ActiveAssets[index].Status = assetStatusFailed
										value.ActiveAssets[index].Status = err.Error()
									} else {
										log.Info(fmt.Sprintf("Applying resources: %v", m.Resources()))
										err = m.Apply()
										if err != nil {
											// Update the asset status with the error message
											log.Error(err, "Error installing the resource", "resource", asset.Name)
											value.ActiveAssets[index].Status = assetStatusFailed
											value.ActiveAssets[index].StatusMessage = err.Error()
										} else {
											value.ActiveAssets[index].Status = assetStatusActive
											value.ActiveAssets[index].StatusMessage = ""
										}
									}
								}
							}
						}
					}
				} else {
					// Add owner reference
					ownerRefs := u.GetOwnerReferences()
					foundOurselves := false
					for _, ownerRef := range ownerRefs {
						if ownerRef.UID == assetOwner.UID {
							foundOurselves = true
						}
					}

					if foundOurselves == false {

						// There can only be one 'controller' reference, so additional references should not
						// be controller references.  It's not clear what Kubernetes does with this field.
						ownerRefs = append(ownerRefs, assetOwner)
						u.SetOwnerReferences(ownerRefs)

						err = c.Update(context.TODO(), u)
						if err != nil {
							log.Error(err, fmt.Sprintf("Unable to add owner reference to %v", asset.Name))
						}
					}

					value.ActiveAssets[index].Status = assetStatusActive
					value.ActiveAssets[index].StatusMessage = ""
				}
			}
		}
	}

	// Now update the StackStatus to reflect the current state of things.
	newStackStatus := kabanerov1alpha2.StackStatus{}
	for i, curSpec := range stackResource.Spec.Versions {
		newStackVersionStatus := kabanerov1alpha2.StackVersionStatus{Version: curSpec.Version}
		if !strings.EqualFold(curSpec.DesiredState, kabanerov1alpha2.StackDesiredStateInactive) {
			if (len(curSpec.DesiredState) > 0) && (!strings.EqualFold(curSpec.DesiredState, kabanerov1alpha2.StackDesiredStateActive)) {
				newStackVersionStatus.StatusMessage = "An invalid desiredState value of " + curSpec.DesiredState + " was specified. The stack is activated by default."
			}
			newStackVersionStatus.Status = kabanerov1alpha2.StackDesiredStateActive

			for _, pipeline := range curSpec.Pipelines {
				key := pipelineUseMapKey{digest: pipeline.Sha256}
				if pipeline.GitRelease.IsUsable() {
					key.gitRelease = gitReleaseSpecToGitReleaseInfo(pipeline.GitRelease)
				} else {
					key.url = pipeline.Https.Url
				}
				value := assetUseMap[key]
				if value == nil {
					// TODO: ???
				} else {
					newStatus := kabanerov1alpha2.PipelineStatus{}
					value.DeepCopyInto(&newStatus)
					newStatus.Name = pipeline.Id // This may vary by stack version
					newStackVersionStatus.Pipelines = append(newStackVersionStatus.Pipelines, newStatus)
					// If we had a problem loading the pipeline manifests, say so.
					if value.manifestError != nil {
						newStackVersionStatus.StatusMessage = value.manifestError.Error()
					}
				}
			}

			// Before we update the status, validate that the images reported in the status do not contain a tag.
			// This action should never need to update the images and it should never fail.
			// If it fails, the stack mutating webhook and/or kabanero stack create/update
			// processing is incorrect.
			err := sutils.RemoveTagFromStackImages(&curSpec, stackResource.Spec.Name)
			if err != nil {
				return err
			}
			stackResource.Spec.Versions[i] = curSpec

			// Update the status of the Stack object to reflect the images used
			for _, img := range curSpec.Images {
				digest := getStatusImageDigest(c, *stackResource, curSpec, img.Image, logger)
				newStackVersionStatus.Images = append(newStackVersionStatus.Images, kabanerov1alpha2.ImageStatus{Id: img.Id, Image: img.Image, Digest: digest})
			}
		} else {
			newStackVersionStatus.Status = kabanerov1alpha2.StackDesiredStateInactive
			newStackVersionStatus.StatusMessage = "The stack has been deactivated."
		}

		log.Info(fmt.Sprintf("Updated stack status: %+v", newStackVersionStatus))
		newStackStatus.Versions = append(newStackStatus.Versions, newStackVersionStatus)
	}

	newStackStatus.Summary = stackSummary(newStackStatus)

	stackResource.Status = newStackStatus

	return nil
}

func getStackForSpecVersion(spec kabanerov1alpha2.StackVersion, stacks []resolvedStack) *resolvedStack {
	for _, stack := range stacks {
		if stack.stack.Version == spec.Version {
			return &stack
		}
	}
	return nil
}

// Retrieves stack image version activation digests. As such, the digest is only captured once during the actvation
// of the stacks. If there is an error during first retrieval, a subsequent successful retry may set the current digest and
// not the activation digest. More precisely, the digest may not necessarily be the initial activation digest
// because we allow stack activation despite there being a failure when retrieving the digest and the
// image/digest may have changed before the next successful retry.
func getStatusImageDigest(c client.Client, stackResource kabanerov1alpha2.Stack, curSpec kabanerov1alpha2.StackVersion, targetImg string, logger logr.Logger) kabanerov1alpha2.ImageDigest {
	digest := kabanerov1alpha2.ImageDigest{}
	foundTargetImage := false

	// If the activation digest was already set, capture its value.
	for _, ssv := range stackResource.Status.Versions {
		if ssv.Version != curSpec.Version {
			continue
		}
		for _, ssvi := range ssv.Images {
			if targetImg != ssvi.Image {
				continue
			}
			if len(ssvi.Digest.Activation) != 0 {
				digest = ssvi.Digest
			}
			foundTargetImage = true
			break
		}
		if foundTargetImage {
			break
		}
	}

	// If the activation digest was not set, find it.
	if digest == (kabanerov1alpha2.ImageDigest{}) {
		digest.Message = ""
		img := targetImg + ":" + curSpec.Version
		registry, err := sutils.GetImageRegistry(img)
		if err != nil {
			digest.Message = fmt.Sprintf("Unable to parse registry from image: %v. Associated stack: %v %v. Error: %v", img, stackResource.Spec.Name, curSpec.Version, err)
		} else {
			imgDig, err := retrieveImageDigest(c, stackResource.GetNamespace(), registry, curSpec.SkipRegistryCertVerification, logger, img)
			if err != nil {
				digest.Message = fmt.Sprintf("Unable to retrieve stack activation digest for image: %v. Associated stack: %v %v. Error: %v", img, stackResource.Spec.Name, curSpec.Version, err)
			} else {
				digest.Activation = imgDig
			}
		}
	}

	return digest
}

// Retrieves the input image digest from the hosting repository.
func retrieveImageDigest(c client.Client, namespace string, imgRegistry string, skipCertVerification bool, logr logr.Logger, image string) (string, error) {
	// Search all secrets under the given namespace for the one containing the required hostname.
	annotationKey := "kabanero.io/docker-"
	secret, err := cutils.GetMatchingSecret(c, namespace, sutils.SecretAnnotationFilter, imgRegistry, annotationKey)
	if err != nil {
		newError := fmt.Errorf("Unable to find secret matching annotation values: %v and %v in namespace %v Error: %v", annotationKey, imgRegistry, namespace, err)
		return "", newError
	}

	// If a secret was found, retrieve the needed information from it.
	var password []byte
	var username []byte
	if secret != nil {
		logr.Info(fmt.Sprintf("Secret used for image registry access: %v. Secret annotations: %v", secret.GetName(), secret.Annotations))
		username, _ = secret.Data["username"]
		password, _ = secret.Data["password"]
	}

	// Create the authenticator mechanism to use for authentication.
	authenticator := authn.Anonymous
	if len(username) != 0 && len(password) != 0 {
		encodedPwd := base64.StdEncoding.EncodeToString([]byte(password))
		decodedPwdBytes, err := base64.StdEncoding.DecodeString(encodedPwd)
		if err != nil {
			return "", err
		}

		encodedUname := base64.StdEncoding.EncodeToString([]byte(username))
		decodedUnameBytes, err := base64.StdEncoding.DecodeString(encodedUname)
		if err != nil {
			return "", err
		}

		authenticator = authn.FromConfig(authn.AuthConfig{
			Username: string(decodedUnameBytes),
			Password: string(decodedPwdBytes)})
	}

	// Retrieve the image manifest.
	ref, err := name.ParseReference(image, name.WeakValidation)
	if err != nil {
		return "", err
	}

	transport := &http.Transport{}
	if skipCertVerification {
		tlsConf := &tls.Config{InsecureSkipVerify: skipCertVerification}
		transport.TLSClientConfig = tlsConf
	}

	img, err := remote.Image(ref,
		remote.WithAuth(authenticator),
		remote.WithPlatform(v1.Platform{Architecture: runtime.GOARCH, OS: runtime.GOOS}),
		remote.WithTransport(transport))
	if err != nil {
		return "", err
	}

	// Get the image's Digest (i.e sha256:8f095a6e...)
	h, err := img.Digest()
	if err != nil {
		return "", err
	}

	// Return the actual digest part only.
	return h.Hex, nil
}

// Drives stack instance deletion processing. This includes creating a finalizer, handling
// stack instance cleanup logic, and finalizer removal.
func processDeletion(ctx context.Context, stack *kabanerov1alpha2.Stack, c client.Client, reqLogger logr.Logger) (bool, error) {
	// The stack instance is not deleted. Create a finalizer if it was not created already.
	stackFinalizer := "kabanero.io/stack-controller"
	foundFinalizer := false
	for _, finalizer := range stack.Finalizers {
		if finalizer == stackFinalizer {
			foundFinalizer = true
		}
	}

	beingDeleted := !stack.DeletionTimestamp.IsZero()
	if !beingDeleted {
		if !foundFinalizer {
			stack.Finalizers = append(stack.Finalizers, stackFinalizer)
			err := c.Update(ctx, stack)
			if err != nil {
				reqLogger.Error(err, "Unable to set the stack controller finalizer.")
				return beingDeleted, err
			}
		}

		return beingDeleted, nil
	}

	// The instance is being deleted.
	if foundFinalizer {
		// Drive stack cleanup processing.
		err := cleanup(ctx, stack, c, reqLogger)
		if err != nil {
			reqLogger.Error(err, "Error during cleanup processing.")
			return beingDeleted, err
		}

		// Remove the finalizer entry from the instance.
		var newFinalizerList []string
		for _, finalizer := range stack.Finalizers {
			if finalizer == stackFinalizer {
				continue
			}
			newFinalizerList = append(newFinalizerList, finalizer)
		}

		stack.Finalizers = newFinalizerList
		err = c.Update(ctx, stack)

		if err != nil {
			reqLogger.Error(err, "Error while attempting to remove the finalizer.")
			return beingDeleted, err
		}
	}

	return beingDeleted, nil
}

// Handles the finalizer cleanup logic for the Stack instance.
func cleanup(ctx context.Context, stack *kabanerov1alpha2.Stack, c client.Client, reqLogger logr.Logger) error {
	ownerIsController := false
	assetOwner := metav1.OwnerReference{
		APIVersion: stack.APIVersion,
		Kind:       stack.Kind,
		Name:       stack.Name,
		UID:        stack.UID,
		Controller: &ownerIsController,
	}

	// Run thru the status and delete everything.... we're just going to try once since it's unlikely
	// that anything that goes wrong here would be rectified by a retry.
	for _, version := range stack.Status.Versions {
		for _, pipeline := range version.Pipelines {
			for _, asset := range pipeline.ActiveAssets {
				// Old assets may not have a namespace set - correct that now.
				if len(asset.Namespace) == 0 {
					asset.Namespace = stack.GetNamespace()
				}

				deleteAsset(c, asset, assetOwner)
			}
		}
	}

	return nil
}

// Deletes an asset.  This can mean removing an object owner, or completely deleting it.
func deleteAsset(c client.Client, asset kabanerov1alpha2.RepositoryAssetStatus, assetOwner metav1.OwnerReference) error {
	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   asset.Group,
		Version: asset.Version,
		Kind:    asset.Kind,
	})

	err := c.Get(context.Background(), client.ObjectKey{
		Namespace: asset.Namespace,
		Name:      asset.Name,
	}, u)

	if err != nil {
		if errors.IsNotFound(err) == false {
			log.Error(err, fmt.Sprintf("Unable to check asset name %v", asset.Name))
			return err
		}
	} else {
		// Get the owner references.  See if we're the last one.
		ownerRefs := u.GetOwnerReferences()
		newOwnerRefs := []metav1.OwnerReference{}
		for _, ownerRef := range ownerRefs {
			if ownerRef.UID != assetOwner.UID {
				newOwnerRefs = append(newOwnerRefs, ownerRef)
			}
		}

		if len(newOwnerRefs) == 0 {
			err = c.Delete(context.TODO(), u)
			if err != nil {
				log.Error(err, fmt.Sprintf("Unable to delete asset name %v", asset.Name))
				return err
			}
		} else {
			u.SetOwnerReferences(newOwnerRefs)
			err = c.Update(context.TODO(), u)
			if err != nil {
				log.Error(err, fmt.Sprintf("Unable to delete owner reference from %v", asset.Name))
				return err
			}
		}
	}

	return nil
}
