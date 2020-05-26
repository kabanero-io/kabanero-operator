package stack

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/docker/cli/cli/config"
	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/cli/cli/config/types"
	"github.com/go-logr/logr"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	kabanerov1alpha2 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha2"
	sutils "github.com/kabanero-io/kabanero-operator/pkg/controller/stack/utils"
	cutils "github.com/kabanero-io/kabanero-operator/pkg/controller/utils"
	"github.com/kabanero-io/kabanero-operator/pkg/controller/utils/secret"

	"github.com/docker/docker/registry"
	pipelinev1alpha1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8runtime "k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
	imagev1 "github.com/openshift/api/image/v1"
)

var log = logf.Log.WithName("controller_stack")
var cIDRegex = regexp.MustCompile("^[a-z]([a-z0-9-]*[a-z0-9])?$")

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

	// Index ImageStreams by status.publicDockerImageRepository
	if err := mgr.GetFieldIndexer().IndexField(&imagev1.ImageStream{}, "status.publicDockerImageRepository", func(rawObj k8runtime.Object) []string {
		imagestream := rawObj.(*imagev1.ImageStream)
		return []string{imagestream.Status.PublicDockerImageRepository }
	}); err != nil {
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

	// Force a requeue if there are failed stacks.
	// This is likely due to a failed image digest lookup.
	// These should be retried, and since they are hosted outside of Kubernetes.
	_, errorSummary := stackSummary(instance.Status)
	if len(errorSummary) != 0 && (rr.Requeue == false) {
		reqLogger.Info(fmt.Sprintf("An error was detected on one or more versions of stack %v. Error version summary: [%v]. Forcing requeue.", instance.Name, errorSummary))
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
				if asset.Status == cutils.AssetStatusFailed {
					return true
				}
			}
		}
	}
	return false
}

// Creates an stack status summary along with a summary of versions containing errors.
func stackSummary(status kabanerov1alpha2.StackStatus) (string, string) {
	var summary = make([]string, len(status.Versions))
	var errorSummary []string
	for i, version := range status.Versions {
		summary[i] = fmt.Sprintf("%v: %v", version.Version, version.Status)
		if version.Status == kabanerov1alpha2.StackStateError {
			errorSummary = append(errorSummary, fmt.Sprintf("%v", version.Version))
		}
	}
	return fmt.Sprintf("[ %v ]", strings.Join(summary, ", ")), fmt.Sprintf(strings.Join(errorSummary, ", "))
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

	// Activate the pipelines used by this stack.
	assetUseMap, err := cutils.ActivatePipelines(stackResource.Spec, stackResource.Status, stackResource.GetNamespace(), renderingContext, assetOwner, c, logger)

	if err != nil {
		return err
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
				key := cutils.PipelineUseMapKey{Digest: pipeline.Sha256}
				if pipeline.GitRelease.IsUsable() {
					key.GitRelease = gitReleaseSpecToGitReleaseInfo(pipeline.GitRelease)
				} else {
					key.Url = pipeline.Https.Url
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
					if value.ManifestError != nil {
						newStackVersionStatus.StatusMessage = value.ManifestError.Error()
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
				digest, err := getStatusImageDigest(c, *stackResource, curSpec, img.Image, logger)
				if err != nil {
					newStackVersionStatus.Status = kabanerov1alpha2.StackStateError
				}
				newStackVersionStatus.Images = append(newStackVersionStatus.Images, kabanerov1alpha2.ImageStatus{Id: img.Id, Image: img.Image, Digest: digest})
			}
		} else {
			newStackVersionStatus.Status = kabanerov1alpha2.StackDesiredStateInactive
			newStackVersionStatus.StatusMessage = "The stack has been deactivated."
		}

		log.Info(fmt.Sprintf("Updated stack status: %+v", newStackVersionStatus))
		newStackStatus.Versions = append(newStackStatus.Versions, newStackVersionStatus)
	}

	newStackStatus.Summary, _ = stackSummary(newStackStatus)

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
func getStatusImageDigest(c client.Client, stackResource kabanerov1alpha2.Stack, curSpec kabanerov1alpha2.StackVersion, targetImg string, logger logr.Logger) (kabanerov1alpha2.ImageDigest, error) {
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
			return digest, err
		} else {
			imgDig, err := retrieveImageDigest(c, stackResource.GetNamespace(), registry, curSpec.SkipRegistryCertVerification, logger, img)
			if err != nil {
				digest.Message = fmt.Sprintf("Unable to retrieve stack activation digest for image: %v. Associated stack: %v %v. Error: %v", img, stackResource.Spec.Name, curSpec.Version, err)
				return digest, err
			} else {
				digest.Activation = imgDig
			}
		}
	}

	return digest, nil
}

// Retrieves the input image digest from the hosting repository.
func retrieveImageDigest(c client.Client, namespace string, imgRegistry string, skipCertVerification bool, logr logr.Logger, image string) (string, error) {
	imagetocheck := image
	imgRegistrytocheck := imgRegistry
	
	// Check if the image is in the local registry - imagestream using the external route
	// If it is using the local registry, use the SA token for auth
	imagestreamlist := &imagev1.ImageStreamList{}
	err := c.List(context.TODO(), imagestreamlist, client.MatchingFields{"status.publicDockerImageRepository": image})
	if err != nil {
		if !errors.IsNotFound(err) {
			newError := fmt.Errorf("Unable to Get ImageStreamList while searching for image %v: %v", imgRegistrytocheck, err)
			return "", newError
		}
	}

	dsecret := &corev1.Secret{}
	// Should only find one image locally, if so, get the svc repository from Status.DockerImageRepository
	// Use the service account dockercfg secret for auth, there is a matching key for the svc repository
	if len(imagestreamlist.Items) != 0 {
		imagetocheck = imagestreamlist.Items[0].Status.DockerImageRepository
		imgRegistrytocheck, err := sutils.GetImageRegistry(imagetocheck)
		if err != nil {
			newError := fmt.Errorf("Unable to parse registry from image: %v. Error: %v", imgRegistrytocheck, err)
			return "", newError
		}
		annotationKey := "kubernetes.io/service-account.name"
		serviceAccount := "kabanero-operator-stack-controller"
		dsecret, err = secret.GetMatchingSecret(c, namespace, sutils.SecretAnnotationFilter, serviceAccount, annotationKey)
		if err != nil {
			newError := fmt.Errorf("Unable to find secret matching annotation values: %v and %v in namespace %v Error: %v", annotationKey, serviceAccount, namespace, err)
			return "", newError
		}
	} else {
	// Otherwise, this is not an internal image
	// Search all secrets under the given namespace for the one containing the annotation with the required hostname.
		annotationKey := "kabanero.io/docker-"
		dsecret, err = secret.GetMatchingSecret(c, namespace, sutils.SecretAnnotationFilter, imgRegistrytocheck, annotationKey)
		if err != nil {
			newError := fmt.Errorf("Unable to find secret matching annotation values: %v and %v in namespace %v Error: %v", annotationKey, imgRegistrytocheck, namespace, err)
			return "", newError
		}
	}

	// If a secret was found, retrieve the needed information from it.
	var password []byte
	var username []byte
	var dockerconfig []byte
	var dockerconfigjson []byte

	if dsecret != nil {
		logr.Info(fmt.Sprintf("Secret used for image registry access: %v. Secret annotations: %v", dsecret.GetName(), dsecret.Annotations))
		username, _ = dsecret.Data[corev1.BasicAuthUsernameKey]
		password, _ = dsecret.Data[corev1.BasicAuthPasswordKey]
		dockerconfig, _ = dsecret.Data[corev1.DockerConfigKey]
		dockerconfigjson, _ = dsecret.Data[corev1.DockerConfigJsonKey]
	}

	// Create the authenticator mechanism to use for authentication.
	authenticator := authn.Anonymous
	if len(username) != 0 && len(password) != 0 {
		authenticator, err = getBasicSecAuth(username, password)
		if err != nil {
			return "", err
		}
	} else if len(dockerconfig) != 0 || len(dockerconfigjson) != 0 {
		authenticator, err = getDockerCfgSecAuth(dockerconfigjson, dockerconfig, imgRegistrytocheck, logr)
		if err != nil {
			return "", err
		}
	}

	// Retrieve the image manifest.
	ref, err := name.ParseReference(imagetocheck, name.WeakValidation)
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

// Returns an authenticator object containing basic authentication credentials.
func getBasicSecAuth(username []byte, password []byte) (authn.Authenticator, error) {
	authenticator := authn.FromConfig(authn.AuthConfig{
		Username: string(username),
		Password: string(password)})

	return authenticator, nil
}

// Returns an authenticator object containing docker config credentials.
// It handles both legacy .dockercfg file data and docker.json file data.
func getDockerCfgSecAuth(dockerconfigjson []byte, dockerconfig []byte, imgRegistry string, reqLogger logr.Logger) (authn.Authenticator, error) {
	// Read the docker config data into a configFile object.
	var dcf *configfile.ConfigFile
	if len(dockerconfigjson) != 0 {
		cf, err := config.LoadFromReader(strings.NewReader(string(dockerconfigjson)))
		if err != nil {
			return nil, fmt.Errorf(fmt.Sprintf("Unable to load/map docker config data. Error: %v", err))
		}
		dcf = cf
	} else {
		cf, err := config.LegacyLoadFromReader(strings.NewReader(string(dockerconfig)))
		if err != nil {
			return nil, fmt.Errorf(fmt.Sprintf("Unable to load/map legacy docker config data. Error: %v", err))
		}
		dcf = cf
	}

	// Resolve the key that will be used to search for the server name entry in the docker config data.
	key := resolveDockerConfRegKey(imgRegistry)

	// If the docker config entry in the secret does not have an authentication entry, default
	// to Anonymous authentication.
	if !dcf.ContainsAuth() {
		reqLogger.Info(fmt.Sprintf("Security credentials for server name: %v could not be found. The docker config data did not contain any authentication information.", key))
		return authn.Anonymous, nil
	}

	// Get the security credentials for the given key (servername).
	// The credentials are obtained from the credential store if one setup/configured; otherwise, they are obtained
	// from the docker config data that was read.
	// Note that it is very important that if the image being read contains the registry name as prefix,
	// the registry name must match the server name used when the docker login was issued. For example, if
	// private server: mysevername:5000 is used when issuing a docker login command, it is expected
	// that the part of the image representing the registry should be mysevername:5000 (i.e.
	// mysevername:5000/path/my-image:1.0.0)
	cfg, err := dcf.GetAuthConfig(key)
	if err != nil {
		return nil, fmt.Errorf("Unable to retrieve credentials from credentials for server name: Key: %v, Error: %v", key, err)
	}

	// No match was found for the server name key. Default to anonymous authentication.
	if cfg == (types.AuthConfig{}) {
		reqLogger.Info(fmt.Sprintf("Security credentials for server name: %v could not be found. The credential store or docker config data did not contain the security credentials for the mentioned server.", key))
		return authn.Anonymous, nil
	}

	// Security credentials were found.
	authenticator := authn.FromConfig(authn.AuthConfig{
		Username:      cfg.Username,
		Password:      cfg.Password,
		Auth:          cfg.Auth,
		IdentityToken: cfg.IdentityToken,
		RegistryToken: cfg.RegistryToken,
	})

	return authenticator, nil
}

// Resolve the server name key to be used when searching for the server name entry in the
// the docker config data or the credential store.
func resolveDockerConfRegKey(imgRegistry string) string {
	var key string
	switch imgRegistry {
	// Docker registry: When logging in to the docker registry, the server name can be either:
	// nothing, docker.io, index.docker.io, or registry-1.docker.io.
	// They are all translated to: https://index.docker.io/v1/ as the server name.
	case registry.IndexName, registry.IndexHostname, registry.DefaultV2Registry.Hostname():
		key = registry.IndexServer
	default:
		key = imgRegistry
	}

	return key
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

				cutils.DeleteAsset(c, asset, assetOwner, reqLogger)
			}
		}
	}

	return nil
}
