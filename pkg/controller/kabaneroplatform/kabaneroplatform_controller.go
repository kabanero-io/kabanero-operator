package kabaneroplatform

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/go-logr/logr"
	kabanerov1alpha2 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha2"
  "github.com/kabanero-io/kabanero-operator/pkg/controller/utils/timer"
	"github.com/kabanero-io/kabanero-operator/pkg/versioning"
	mfc "github.com/manifestival/controller-runtime-client"
	mf "github.com/manifestival/manifestival"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
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

var log = logf.Log.WithName("controller_kabaneroplatform")
var ctrlr controller.Controller
var operatorContainerImage string
var operatorContainerImageOp sync.Once

// A list of functions driven by the reconciler
type reconcileFunc func(context.Context, *kabanerov1alpha2.Kabanero, client.Client, logr.Logger) error

type reconcileFuncType struct {
	name     string
	function reconcileFunc
}

var reconcileFuncs = []reconcileFuncType{
	{name: "stack controller", function: reconcileStackController},
	{name: "landing page", function: deployLandingPage},
	{name: "cli service", function: reconcileKabaneroCli},
	{name: "CodeReady Workspaces", function: reconcileCRW},
	{name: "events", function: reconcileEvents},
	{name: "sso", function: reconcileSso},
	{name: "gitops", function: reconcileGitopsPipelines},
	{name: "target namespaces", function: reconcileTargetNamespaces},
	{name: "serving", function: reconcileServing},
}

// Add creates a new Kabanero Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	// It is very unlikely that this would fail, since the main also checks for it.
	watchNamespace, err := k8sutil.GetWatchNamespace()
	if err != nil {
		return err
}

	// Lets be sure a single namespace is specified.
	numberOfWatchNamespaces := len(strings.Split(watchNamespace, ","))
	if numberOfWatchNamespaces != 1 {
		return fmt.Errorf("%v watch namespaces were specified, but only a single watch namespace is supported: %v", numberOfWatchNamespaces, watchNamespace)
	}
	
	r := &ReconcileKabanero{
		client:          mgr.GetClient(),
		scheme:          mgr.GetScheme(),
		requeueDelayMap: make(map[string]RequeueData),
	  watchNamespace:  watchNamespace}

	// Create a new controller
	c, err := controller.New("kabaneroplatform-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}
	ctrlr = c

	// Watch for changes to primary resource Kabanero
	err = c.Watch(&source.Kind{Type: &kabanerov1alpha2.Kabanero{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch Stacks
	err = c.Watch(&source.Kind{Type: &kabanerov1alpha2.Stack{}}, getWatchHandlerForKabaneroOwner(), getWatchPredicateFunc())
	if err != nil {
		return err
	}

	// Watch Kabanero owned deployments.
	err = c.Watch(&source.Kind{Type: &appsv1.Deployment{}}, getWatchHandlerForKabaneroOwner(), getWatchPredicateFunc())
	if err != nil {
		return err
	}

	// Watch CheCluster instances.  We watch these so that we can enforce
	// some fields that should not be changed by the user.
	err = watchCRWInstance(c)
	if err != nil {
		return err
	}

	// Watch Namespace instances.  We only care about create and delete events, not update events.
	// When we see that a namespace has been created/deleted, we need to process any Kabanero objects that
	// reference that namespace.
	err = c.Watch(&source.Kind{Type: &corev1.Namespace{}}, &handler.EnqueueRequestsFromMapFunc{
		ToRequests: handler.ToRequestsFunc(r.targetNamespaceMapFunc)}, predicate.Funcs{
			UpdateFunc: func(e event.UpdateEvent) bool { return false }})
	if err != nil {
		return err
	}

/* Useful if RoleBindingList is changed to use Structured instead of Unstructured
	// Index Rolebindings by name
	if err := mgr.GetFieldIndexer().IndexField(&rbacv1.RoleBinding{}, "metadata.name", func(rawObj runtime.Object) []string {
		rolebinding := rawObj.(*rbacv1.RoleBinding)
		return []string{rolebinding.ObjectMeta.Name}
	}); err != nil {
		return err
	}
*/

	return nil
}

func (r *ReconcileKabanero) getOperatorImage() (string, error) {
	// First, read the POD_NAME env variable.  This is set in the deployment spec in the CSV.
	podName := os.Getenv("POD_NAME")
	if len(podName) == 0 {
		return "", fmt.Errorf("The POD_NAME environment variable is not set, or is empty")
	}

	// Second, get the Pod instance with that name
	pod := &corev1.Pod{}
	kubePodName := types.NamespacedName{Name: podName, Namespace: r.watchNamespace}
	err := r.client.Get(context.TODO(), kubePodName, pod)
	if err != nil {
		return "", fmt.Errorf("Pod %v could not be retrieved: %v", podName, err.Error())
	} 
	
	// Third, parse out the container name, then the image
	for _, container := range pod.Spec.Containers {
		if container.Name == "kabanero-operator" {
			return container.Image, nil
		}
	}

	return "", fmt.Errorf("No container named 'kabanero-operator' was found in Pod %v", podName)
}


// Returns a watch handler.
func getWatchHandlerForKabaneroOwner() *handler.EnqueueRequestForOwner {
	return &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &kabanerov1alpha2.Kabanero{},
	}
}

// Returns a watch predicate.
func getWatchPredicateFunc() predicate.Funcs {
	return predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			// Returning true only when the metadata generation has changed,
			// allows us to ignore events where only the object status has changed,
			// since the generation is not incremented when only the status changes
			return e.MetaOld.GetGeneration() != e.MetaNew.GetGeneration()
		},
	}
}

var _ reconcile.Reconciler = &ReconcileKabanero{}

// ReconcileKabanero reconciles a KabaneroPlatform object
type ReconcileKabanero struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client          client.Client
	scheme          *runtime.Scheme
	requeueDelayMap map[string]RequeueData
	watchNamespace  string
}

// RequeueData stores information that enables reconcile operations to be retried.
type RequeueData struct {
	delay      int
	futureTime time.Time
}

// When we see that a namespace has changed, we want to reconcile any Kabanero instances that
// reference that namespace in its targetNamespaces list.
func (r *ReconcileKabanero) targetNamespaceMapFunc(a handler.MapObject) []reconcile.Request {
	log.Info(fmt.Sprintf("Processing for change in namespace %v", a.Meta.GetName()))
	
	// List Kabanero instances
	kabaneros := &kabanerov1alpha2.KabaneroList{}
	err := r.client.List(context.TODO(), kabaneros, client.InNamespace(r.watchNamespace))
	if err != nil {
		log.Error(err, fmt.Sprintf("Could not process namespace event for \"%v\"", a.Meta.GetName()))
		return nil
	}

	// For each Kabanero instance, if spec.targetNamespaces includes a.meta.name then add a reconcile request.
	requests := []reconcile.Request{}
	for _, kabanero := range kabaneros.Items {
		for _, namespace := range kabanero.Spec.TargetNamespaces {
			if namespace == a.Meta.GetName() {
				requests = append(requests, reconcile.Request{types.NamespacedName{Name: kabanero.Name, Namespace: kabanero.Namespace}})
				break
			}
		}
	}
	
  return requests
}

// Determine if requeue is needed or not.
// If requeue is required set RequeueAfter to 60 seconds the first time.
// After the first time increase RequeueAfter by 60 seconds up to a max of 15 minutes.
func (r *ReconcileKabanero) determineHowToRequeue(ctx context.Context, request reconcile.Request, instance *kabanerov1alpha2.Kabanero, errorMessage string, requeueDelayMap map[string]RequeueData, reqLogger logr.Logger) (reconcile.Result, error) {
	var requeueDelay int
	var localFutureTime time.Time
	requeueData, ok := requeueDelayMap[request.Namespace]
	if !ok {
		requeueDelay = 0
		localFutureTime = time.Now()
		requeueData := RequeueData{requeueDelay, localFutureTime}
		requeueDelayMap[request.Namespace] = requeueData
	} else {
		requeueDelay = requeueData.delay
		localFutureTime = requeueData.futureTime
	}
	currentTime := time.Now()
	// if first time or current time is after the requeue time we requested previously, request a requeue
	if requeueDelay == 0 || currentTime.After(localFutureTime) {
		// only update status if error message changed
		if strings.Compare(errorMessage, instance.Status.KabaneroInstance.Message) != 0 {
			instance.Status.KabaneroInstance.Message = errorMessage
			instance.Status.KabaneroInstance.Ready = "False"
			// Update the kabanero instance status.
			err := r.client.Status().Update(ctx, instance)
			if err != nil {
				reqLogger.Error(err, "Error updating Kabanero status.")
			}
		}
		// increase delay by 60 seconds
		requeueDelay = requeueDelay + 60
		// do not go over 15 minutes
		if requeueDelay >= 900 {
			requeueDelay = 900
		}
		localFutureTime = currentTime.Add(time.Duration(requeueDelay) * time.Second)
		requeueDelayMap[request.Namespace] = RequeueData{requeueDelay, localFutureTime}
		reqLogger.Info(fmt.Sprintf("Reconciling Kabanero requesting requeue in %d seconds", requeueDelay))

		return reconcile.Result{Requeue: true, RequeueAfter: time.Duration(requeueDelay) * time.Second}, nil
	}

	// No requeue
	return reconcile.Result{}, nil

}



// Reconcile reads that state of the cluster for a Kabanero object and makes changes based on the state read
// and what is in the Kabanero.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileKabanero) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	ctx := context.Background()

	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling Kabanero")

	// Retrieve the Kabanero operator image name, for use later.  Only do this once.  Can't do it
	// in the add() method because the client is not started yet (that would have been ideal).
	operatorContainerImageOp.Do(func() {
		var err error
		operatorContainerImage, err = r.getOperatorImage()
		if err != nil {
			log.Error(err, "Could not read the kabanero-operator container image from the pod")
		}
	})

	// TODO: Retrieve kabanero as unstructured
	kabInstanceUnstructured := &unstructured.Unstructured{}
	kabInstanceUnstructured.SetGroupVersionKind(schema.GroupVersionKind{
		Kind:    "Kabanero",
		Group:   kabanerov1alpha2.SchemeGroupVersion.Group,
		Version: kabanerov1alpha2.SchemeGroupVersion.Version,
	})

	err := r.client.Get(context.TODO(), request.NamespacedName, kabInstanceUnstructured)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}


	// Fetch the Kabanero instance
	instance := &kabanerov1alpha2.Kabanero{}
	err = r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// Initializes dependency data
	initializeDependencies(instance)

	// Process kabanero instance deletion logic.
	beingDeleted, err := processDeletion(ctx, instance, r.client, reqLogger)
	if err != nil {
		return reconcile.Result{}, err
	}

	if beingDeleted {
		return reconcile.Result{}, nil
	}

	// Reconcile the admission controller webhook
	err = reconcileAdmissionControllerWebhook(ctx, instance, r.client, reqLogger)
	if err != nil {
		reqLogger.Error(err, "Error reconciling kabanero-admission-controller-webhook")
		return reconcile.Result{}, err
	}

	// Collections are no longer supported.  Remove any objects that were used by
	// the collection controller.
	cleanupCollectionController(ctx, instance, r.client, reqLogger)
	
	// Wait for the admission controller webhook to be ready before we try
	// to deploy the featured stacks.
	isAdmissionControllerWebhookReady, _ := getAdmissionControllerWebhookStatus(instance, r.client, reqLogger)
	if isAdmissionControllerWebhookReady == false {
		processStatus(ctx, request, instance, r.client, reqLogger)
		return reconcile.Result{Requeue: true, RequeueAfter: 10 * time.Second}, nil
	}

	// Iterate the components and try to reconcile.  If something goes wrong,
	// update the status and try again later.
	for _, component := range reconcileFuncs {
		err = component.function(ctx, instance, r.client, reqLogger)
		if err != nil {
			reqLogger.Error(err, fmt.Sprintf("Error deploying %v.", component.name))
			processStatus(ctx, request, instance, r.client, reqLogger)
			return reconcile.Result{}, err
		}
	}

	// Deploy featured stack resources.
	err = reconcileFeaturedStacks(ctx, instance, r.client, reqLogger)
	if err != nil {
		reqLogger.Error(err, "Error reconciling featured stacks.")
		processStatus(ctx, request, instance, r.client, reqLogger)
		return r.determineHowToRequeue(ctx, request, instance, err.Error(), r.requeueDelayMap, reqLogger)
	}

	// things worked reset requeue data
	r.requeueDelayMap[request.Namespace] = RequeueData{0, time.Now()}

	// Determine the status of the kabanero operator instance and set it.
	isReady, err := processStatus(ctx, request, instance, r.client, reqLogger)
	if err != nil {
		reqLogger.Error(err, "Error updating the status.")
		return reconcile.Result{}, err
	}

	// If all resource dependencies are not in the ready state, reconcile again in 60 seconds.
	if !isReady {
		return reconcile.Result{Requeue: true, RequeueAfter: 60 * time.Second}, err
	}

	return reconcile.Result{}, nil
}

// Drives kabanero instance deletion processing. This includes creating a finalizer, handling
// kabanero instance cleanup logic, and finalizer removal.
func processDeletion(ctx context.Context, k *kabanerov1alpha2.Kabanero, client client.Client, reqLogger logr.Logger) (bool, error) {
	// The kabanero instance is not deleted. Create a finalizer if it was not created already.
	kabaneroFinalizer := "kabanero.io.kabanero-operator"
	foundFinalizer := isFinalizerInList(k, kabaneroFinalizer)
	beingDeleted := !k.ObjectMeta.DeletionTimestamp.IsZero()
	if !beingDeleted {
		if !foundFinalizer {
			k.ObjectMeta.Finalizers = append(k.ObjectMeta.Finalizers, kabaneroFinalizer)
			// Need to cache the Group/Version/Kind here because the Update call
			// will clear them.  This is fixed in controller-runtime v0.2.0 /
			// operator-sdk 0.11.0.  TODO
			gvk := k.GroupVersionKind()
			err := client.Update(ctx, k)
			if err != nil {
				reqLogger.Error(err, "Unable to set the kabanero operator finalizer.")
				return beingDeleted, err
			}
			k.SetGroupVersionKind(gvk)
		}

		return beingDeleted, nil
	}

	// The instance is being deleted.
	if foundFinalizer {
		// Drive kabanero cleanup processing.
		err := cleanup(ctx, k, client, reqLogger)
		if err != nil {
			reqLogger.Error(err, "Error during cleanup processing.")
			return beingDeleted, err
		}

		// Remove the finalizer entry from the instance.
		var newFinalizerList []string
		for _, finalizer := range k.ObjectMeta.Finalizers {
			if finalizer == kabaneroFinalizer {
				continue
			}
			newFinalizerList = append(newFinalizerList, finalizer)
		}

		k.ObjectMeta.Finalizers = newFinalizerList

		// Need to cache the Group/Version/Kind here because the Update call
		// will clear them.  This is fixed in controller-runtime v0.2.0 /
		// operator-sdk 0.11.0.  TODO
		gvk := k.GroupVersionKind()
		err = client.Update(ctx, k)

		if err != nil {
			reqLogger.Error(err, "Error while attempting to remove the finalizer.")
			return beingDeleted, err
		}

		k.SetGroupVersionKind(gvk)
	}

	return beingDeleted, nil
}

// Handles all cleanup logic for the Kabanero instance.
func cleanup(ctx context.Context, k *kabanerov1alpha2.Kabanero, client client.Client, reqLogger logr.Logger) error {
	// if landing enabled
	if k.Spec.Landing.Enable == nil || (k.Spec.Landing.Enable != nil && *(k.Spec.Landing.Enable) == true) {
		// Remove landing page customizations for the current namespace.
		err := removeWebConsoleCustomization(k, client)
		if err != nil {
			return err
		}
	}

	// Remove the webhook configurations and friends.
	err := cleanupAdmissionControllerWebhook(k, client, reqLogger)
	if err != nil {
		return err
	}

	// Remove the cross-namespace objects that the stack controller uses.
	err = cleanupStackController(ctx, k, client)
	if err != nil {
		return err
	}

	// Remove resources deployed in support of codeready-workspaces.
	err = deleteCRWOperatorResources(ctx, k, client)
	if err != nil {
		return err
	}

	// Cleanup the Gitops pipelines and their cross-namespace objects
	err = cleanupGitopsPipelines(ctx, k, client, reqLogger)
	if err != nil {
		return err
	}
	
	// Remove the cross-namespace objects that target namespaces use.
	err = cleanupTargetNamespaces(ctx, k, client)
	if err != nil {
		return err
	}
	
	// Cleanup the Serving and their cross-namespace objects
	err = cleanupServing(k, client, reqLogger)
	if err != nil {
		return err
	}

	return nil
}

// Returns true if the kabanero operator instance has the given finalizer defined. False otherwise.
func isFinalizerInList(k *kabanerov1alpha2.Kabanero, finalizer string) bool {
	for _, f := range k.ObjectMeta.Finalizers {
		if f == finalizer {
			return true
		}
	}
	return false
}

// Retrieves Kabanero resource dependencies' readiness status to determine the Kabanero instance readiness status.
// If all resource dependencies are in the ready state, the kabanero instance's readiness status
// is set to true. Otherwise, it is set to false.
func processStatus(ctx context.Context, request reconcile.Request, k *kabanerov1alpha2.Kabanero, c client.Client, reqLogger logr.Logger) (bool, error) {
	errorMessage := "One or more resource dependencies are not ready."
	_, instanceVersion := resolveKabaneroVersion(k)
	k.Status.KabaneroInstance.Version = instanceVersion

	k.Status.KabaneroInstance.Ready = "False"

	// Gather the status of all resource dependencies.
	isStackControllerReady, _ := getStackControllerStatus(ctx, k, c)
	isAppsodyReady, _ := getAppsodyStatus(k, c, reqLogger)
	isTektonReady, _ := getTektonStatus(k, c)
	isServerlessReady, _ := getServerlessStatus(k, c, reqLogger)
	isCliRouteReady, _ := getCliRouteStatus(k, reqLogger, c)
	isKabaneroLandingReady, _ := getKabaneroLandingPageStatus(k, c)
	isKubernetesAppNavigatorReady, _ := getKappnavStatus(k, c)
	isCRWReady, _ := getCRWStatus(ctx, k, c)
	isEventsReady, _ := getEventsStatus(k, c, reqLogger)
	isAdmissionControllerWebhookReady, _ := getAdmissionControllerWebhookStatus(k, c, reqLogger)
	isSsoReady, _ := getSsoStatus(k, c, reqLogger)
	isGitopsReady, _ := getGitopsStatus(k)
	isTargetNamespacesReady, _ := getTargetNamespacesStatus(k)

	// Set the overall status.
	isKabaneroReady := isStackControllerReady &&
		isTektonReady &&
		isServerlessReady &&
		isCliRouteReady &&
		isKabaneroLandingReady &&
		isAppsodyReady &&
		isKubernetesAppNavigatorReady &&
		isCRWReady &&
		isEventsReady &&
		isAdmissionControllerWebhookReady &&
		isSsoReady &&
		isGitopsReady &&
		isTargetNamespacesReady

	if isKabaneroReady {
		k.Status.KabaneroInstance.Message = ""
		k.Status.KabaneroInstance.Ready = "True"
	} else {
		k.Status.KabaneroInstance.Message = errorMessage
	}

	// Update the kabanero instance status in a retriable manner. The instance may have changed.
	err := timer.Retry(10, 100*time.Millisecond, func() (bool, error) {
		err := c.Status().Update(ctx, k)
		if err != nil {
			if errors.IsConflict(err) {
				k = &kabanerov1alpha2.Kabanero{}
				err = c.Get(context.TODO(), request.NamespacedName, k)

				if err != nil {
					return false, err
				}

				return false, nil
			}
		}

		return true, nil
	})

	return isKabaneroReady, err
}

// Initializes dependencies.
func initializeDependencies(k *kabanerov1alpha2.Kabanero) {
	// Codeready-workspaces initialization.
	initializeCRW(k)
}

// Cleanup the collection controller (used in past releases)
func cleanupCollectionController(ctx context.Context, k *kabanerov1alpha2.Kabanero, cl client.Client, reqLogger logr.Logger) {
	// Easiest thing to do is probably to load the orchestration and delete everything.
	orchestrationPath := "orchestrations/collection-controller/0.1"
	templateContext := make(map[string]interface{})
	templateContext["instance"] = "nil"
	templateContext["version"] = "nil"
	templateContext["image"] = "nil:nil"
	templateContext["name"] = "kabanero-" + k.GetNamespace() + "-trigger-rolebinding"
	templateContext["kabaneroNamespace"] = k.GetNamespace()
	rev := versioning.SoftwareRevision{Version: "nil", OrchestrationPath: orchestrationPath, Identifiers: templateContext}

	transformMap := make(map[string][]mf.Transformer)
	transformMap["collection-controller.yaml"] = []mf.Transformer{mf.InjectNamespace(k.GetNamespace())}
	transformMap["collection-controller-tekton.yaml"] = []mf.Transformer{}
	for yaml, transforms := range transformMap {
		f, err := rev.OpenOrchestration(yaml)
		if err != nil {
			reqLogger.Error(err, fmt.Sprintf("Unable to open %v orchestration", yaml))
		} else {
			s, err := renderOrchestration(f, templateContext)
			if err != nil {
				reqLogger.Error(err, fmt.Sprintf("Unable to render %v orchestration", yaml))
			} else {
				m, err := mf.ManifestFrom(mf.Reader(strings.NewReader(s)), mf.UseClient(mfc.NewClient(cl)), mf.UseLogger(reqLogger.WithName("manifestival")))
				if err != nil {
					reqLogger.Error(err, fmt.Sprintf("Unable to load manifests for %v orchestration", yaml))
				} else {
					mt, err := m.Transform(transforms...)
					if err != nil {
						reqLogger.Error(err, fmt.Sprintf("Unable to transform manifests for %v orchestration", yaml))
					} else {
						err = mt.Delete()
						if err != nil {
							reqLogger.Error(err, fmt.Sprintf("Unable to delete %v objects", yaml))
						}
					}
				}
			}
		}
	}
}
