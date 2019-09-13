package kabaneroplatform

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-logr/logr"
	kabanerov1alpha1 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
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
	return &ReconcileKabanero{client: mgr.GetClient(), scheme: mgr.GetScheme(), requeueDelayMapV1: make(map[string]RequeueData), requeueDelayMapV2: make(map[string]RequeueData)}
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

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner Kabanero
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &kabanerov1alpha1.Kabanero{},
	})
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

type RequeueData struct {
	delay      int
	futureTime time.Time
}

// Determine if requeue is needed or not.
// If requeue is required set RequeueAfter to 60 seconds the first time.
// After the first time increase RequeueAfter by 60 seconds up to a max of 15 minutes.
func (r *ReconcileKabanero) determineHowToRequeue(request reconcile.Request, ctx context.Context, instance *kabanerov1alpha1.Kabanero, errorMessage string, requeueDelayMap map[string]RequeueData, reqLogger logr.Logger) (reconcile.Result, error) {
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
	} else { // no requeue
		return reconcile.Result{}, nil
	}
}

// Reconcile reads that state of the cluster for a Kabanero object and makes changes based on the state read
// and what is in the Kabanero.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
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
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	err = reconcileFeaturedCollections(ctx, instance, r.client)
	if err != nil {
		fmt.Println("Error in reconcile featured collections: ", err)
		return r.determineHowToRequeue(request, ctx, instance, err.Error(), r.requeueDelayMapV1, reqLogger)
	}

	// things worked reset requeue data
	r.requeueDelayMapV1[request.Namespace] = RequeueData{0, time.Now()}

	err = reconcileFeaturedCollectionsV2(ctx, instance, r.client)
	if err != nil {
		fmt.Println("Error in reconcile featured collections V2: ", err)
		return r.determineHowToRequeue(request, ctx, instance, err.Error(), r.requeueDelayMapV2, reqLogger)
	}
	// things worked reset requeue data
	r.requeueDelayMapV2[request.Namespace] = RequeueData{0, time.Now()}

	//Reconcile the appsody operator
	err = reconcile_appsody(ctx, instance, r.client)
	if err != nil {
		fmt.Println("Error in reconcile appsody: ", err)
		return reconcile.Result{}, err
	}

	// Deploy the kabanero landing page
	err = deployLandingPage(instance, r.client)
	if err != nil {
		fmt.Println("Error deploying the kabanero landing page: ", err)
		return reconcile.Result{}, err
	}

	// Reconcile the Kabanero CLI.
	err = reconcileKabaneroCli(ctx, instance, r.client, reqLogger)
	if err != nil {
		fmt.Println("Error in reconcile Kabanero CLI: ", err)
		return reconcile.Result{}, err
	}

	// Reconcile the Kubernetes Application Navigator if enabled. It is disabled by default.
	err = reconcileKappnav(ctx, instance, r.client)
	if err != nil {
		fmt.Println("Error reconciling the Kubernetes Application Navigator: ", err)
		return reconcile.Result{}, err
	}

	// Determine the status of the kabanero operator instance and set it.
	isReady, err := processStatus(ctx, instance, r.client, reqLogger)
	if err != nil {
		fmt.Println("Error updating the status: ", err)
		return reconcile.Result{}, err
	}

	// If all resoruce dependencies are not in the ready state, reconcile again in 60 seconds.
	if !isReady {
		return reconcile.Result{Requeue: true, RequeueAfter: 60 * time.Second}, err
	}

	return reconcile.Result{}, nil
}

// Retrieves Kabanero resource dependencies' readiness status to determine the Kabanero instance readiness status.
// If all resource dependencies are in the ready state, the kabanero instance's readiness status
// is set to true. Otherwise, it is set to false.
func processStatus(ctx context.Context, k *kabanerov1alpha1.Kabanero, c client.Client, reqLogger logr.Logger) (bool, error) {
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

	// Update the kabanero instance status.
	err := c.Status().Update(ctx, k)
	if err != nil {
		fmt.Println("Error updating the status.", err)
	}

	return isKabaneroReady, err
}
