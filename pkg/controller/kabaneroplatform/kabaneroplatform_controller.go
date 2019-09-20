package kabaneroplatform

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-logr/logr"
	kabanerov1alpha1 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha1"
	kutils "github.com/kabanero-io/kabanero-operator/pkg/controller/kabaneroplatform/utils"

	//	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_kabaneroplatform")

// Add creates a new Kabanero Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileKabanero{
		client:            mgr.GetClient(),
		scheme:            mgr.GetScheme(),
		requeueDelayMapV1: make(map[string]RequeueData),
		requeueDelayMapV2: make(map[string]RequeueData)}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("kabaneroplatform-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Kabanero
	err = c.Watch(&source.Kind{Type: &kabanerov1alpha1.Kabanero{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Create a handler
	tH := &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &kabanerov1alpha1.Kabanero{},
	}

	// Create predicate
	tPred := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			// Returning true only when the metadata generation has changed,
			// allows us to ignore events where only the object status has changed,
			// since the generation is not incremented when only the status changes
			return e.MetaOld.GetGeneration() != e.MetaNew.GetGeneration()
		},
	}

	//Watch Collections
	err = c.Watch(&source.Kind{Type: &kabanerov1alpha1.Collection{}}, tH, tPred)
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileKabanero{}

// ReconcileKabanero reconciles a KabaneroPlatform object
type ReconcileKabanero struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client            client.Client
	scheme            *runtime.Scheme
	requeueDelayMapV1 map[string]RequeueData
	requeueDelayMapV2 map[string]RequeueData
}

// RequeueData stores information that enables reconcile operations to be retried.
type RequeueData struct {
	delay      int
	futureTime time.Time
}

// Determine if requeue is needed or not.
// If requeue is required set RequeueAfter to 60 seconds the first time.
// After the first time increase RequeueAfter by 60 seconds up to a max of 15 minutes.
func (r *ReconcileKabanero) determineHowToRequeue(ctx context.Context, request reconcile.Request, instance *kabanerov1alpha1.Kabanero, errorMessage string, requeueDelayMap map[string]RequeueData, reqLogger logr.Logger) (reconcile.Result, error) {
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
		if strings.Compare(errorMessage, instance.Status.KabaneroInstance.ErrorMessage) != 0 {
			instance.Status.KabaneroInstance.ErrorMessage = errorMessage
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

	// Fetch the Kabanero instance
	instance := &kabanerov1alpha1.Kabanero{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
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

	// Process kabanero instance deletion logic.
	beingDeleted, err := processDeletion(ctx, instance, r.client, reqLogger)
	if err != nil {
		return reconcile.Result{}, err
	}

	if beingDeleted {
		return reconcile.Result{}, nil
	}

	// Deploy version 1 feature collection resources.
	err = reconcileFeaturedCollections(ctx, instance, r.client)
	if err != nil {
		reqLogger.Error(err, "Error reconciling featured collections.")
		return r.determineHowToRequeue(ctx, request, instance, err.Error(), r.requeueDelayMapV1, reqLogger)
	}

	// things worked reset requeue data
	r.requeueDelayMapV1[request.Namespace] = RequeueData{0, time.Now()}

	// Deploy version 2 feature collection resources.
	err = reconcileFeaturedCollectionsV2(ctx, instance, r.client)
	if err != nil {
		reqLogger.Error(err, "Error reconciling featured collections V2.")
		return r.determineHowToRequeue(ctx, request, instance, err.Error(), r.requeueDelayMapV2, reqLogger)
	}
	// things worked reset requeue data
	r.requeueDelayMapV2[request.Namespace] = RequeueData{0, time.Now()}

	// Reconcile the appsody operator
	err = reconcile_appsody(ctx, instance, r.client)
	if err != nil {
		reqLogger.Error(err, "Error reconciling appsody.")
		return reconcile.Result{}, err
	}

	// Deploy the kabanero landing page
	err = deployLandingPage(instance, r.client)
	if err != nil {
		reqLogger.Error(err, "Error deploying the kabanero landing page.")
		return reconcile.Result{}, err
	}

	// Reconcile the Kabanero CLI.
	err = reconcileKabaneroCli(ctx, instance, r.client, reqLogger)
	if err != nil {
		reqLogger.Error(err, "Error reconciling the Kabanero CLI.")
		return reconcile.Result{}, err
	}

	// Reconcile the Kubernetes Application Navigator if enabled. It is disabled by default.
	err = reconcileKappnav(ctx, instance, r.client)
	if err != nil {
		reqLogger.Error(err, "Error reconciling the Kubernetes Application Navigator.")
		return reconcile.Result{}, err
	}

	// Determine the status of the kabanero operator instance and set it.
	isReady, err := processStatus(ctx, request, instance, r.client, reqLogger)
	if err != nil {
		reqLogger.Error(err, "Error updating the status.")
		return reconcile.Result{}, err
	}

	// If all resoruce dependencies are not in the ready state, reconcile again in 60 seconds.
	if !isReady {
		return reconcile.Result{Requeue: true, RequeueAfter: 60 * time.Second}, err
	}

	return reconcile.Result{}, nil
}

// Drives kabanero instance deletion processing. This includes creating a finalizer, handling
// kabanero instance cleanup logic, and finalizer removal.
func processDeletion(ctx context.Context, k *kabanerov1alpha1.Kabanero, c client.Client, reqLogger logr.Logger) (bool, error) {
	// The kabanero instance is not deleted. Create a finalizer if it was not created already.
	kabaneroFinalizer := "kabanero.io.kabanero-operator"
	foundFinalizer := isFinalizerInList(k, kabaneroFinalizer)
	beingDeleted := !k.ObjectMeta.DeletionTimestamp.IsZero()

	if !beingDeleted {
		if !foundFinalizer {
			k.ObjectMeta.Finalizers = append(k.ObjectMeta.Finalizers, kabaneroFinalizer)
			err := c.Update(ctx, k)
			if err != nil {
				reqLogger.Error(err, "Unable to set the kabanero operator finalizer.")
				return beingDeleted, err
			}
		}

		return beingDeleted, nil
	}

	// The instance is being deleted.
	if foundFinalizer {
		// Drive kabanero cleanup processing.
		err := cleanup(ctx, k, c)
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
		err = c.Update(ctx, k)

		if err != nil {
			reqLogger.Error(err, "Error while attempting to remove the finalizer.")
			return beingDeleted, err
		}
	}

	return beingDeleted, nil
}

// Handles all cleanup logic for the Kabanero instance.
func cleanup(ctx context.Context, k *kabanerov1alpha1.Kabanero, client client.Client) error {
	// Remove landing page customizations for the current namespace.
	err := removeWebConsoleCustomization(k)
	if err != nil {
		return err
	}

	// Delete the KAppNav instance if enabled. Other kAppNav resource will be deleted automatically
	// when the kabanero instance ends.
	if k.Spec.Kappnav.Enable {
		err = deleteKappnavInstance(ctx, k, client)
		if err != nil {
			return err
		}
	}

	return nil
}

// Returns true if the kabanero operator instance has the given finalizer defined. False otherwise.
func isFinalizerInList(k *kabanerov1alpha1.Kabanero, finalizer string) bool {
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
func processStatus(ctx context.Context, request reconcile.Request, k *kabanerov1alpha1.Kabanero, c client.Client, reqLogger logr.Logger) (bool, error) {
	errorMessage := "One or more resource dependencies are not ready."
	//k.Status.KabaneroInstance.Version = version.Version
	k.Status.KabaneroInstance.Ready = "False"

	// Gather the status of all resource dependencies.
	isAppsodyReady, _ := getAppsodyStatus(k, c, reqLogger)
	isTektonReady, _ := getTektonStatus(k, c)
	isKnativeEventingReady, _ := getKnativeServingStatus(k, c)
	isKnativeServingReady, _ := getKnativeEventingStatus(k, c)
	isCliRouteReady, _ := getCliRouteStatus(k, reqLogger)
	isKabaneroLandingReady, _ := getKabaneroLandingPageStatus(k, c)
	isKubernetesAppNavigatorReady, _ := getKappnavStatus(k, c)

	// Set the overall status.
	isKabaneroReady := isTektonReady &&
		isKnativeEventingReady &&
		isKnativeServingReady &&
		isCliRouteReady &&
		isKabaneroLandingReady &&
		isAppsodyReady &&
		isKubernetesAppNavigatorReady
	if isKabaneroReady {
		k.Status.KabaneroInstance.ErrorMessage = ""
		k.Status.KabaneroInstance.Ready = "True"
	} else {
		k.Status.KabaneroInstance.ErrorMessage = errorMessage
	}

	// Update the kabanero instance status in a retriable manner. The instance may have changed.
	err := kutils.Retry(10, 100*time.Millisecond, func() (bool, error) {
		err := c.Status().Update(ctx, k)
		if err != nil {
			if errors.IsConflict(err) {
				k = &kabanerov1alpha1.Kabanero{}
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
