package knativeeventing

import (
	"context"
	"flag"
	"fmt"

	mf "github.com/jcrossley3/manifestival"
	eventingv1alpha1 "github.com/openshift-knative/knative-eventing-operator/pkg/apis/eventing/v1alpha1"
	"github.com/openshift-knative/knative-eventing-operator/version"
	"github.com/operator-framework/operator-sdk/pkg/predicate"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	operand = "knative-eventing"
)

var (
	filename = flag.String("filename", "deploy/resources",
		"The filename containing the YAML resources to apply")
	recursive = flag.Bool("recursive", false,
		"If filename is a directory, process all manifests recursively")
	log = logf.Log.WithName("controller_knativeeventing")
)

// Add creates a new KnativeEventing Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileKnativeEventing{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("knativeeventing-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource KnativeEventing
	err = c.Watch(&source.Kind{Type: &eventingv1alpha1.KnativeEventing{}}, &handler.EnqueueRequestForObject{}, predicate.GenerationChangedPredicate{})
	if err != nil {
		return err
	}

	// Watch child deployments for availability
	err = c.Watch(&source.Kind{Type: &appsv1.Deployment{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &eventingv1alpha1.KnativeEventing{},
	})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileKnativeEventing{}

// ReconcileKnativeEventing reconciles a KnativeEventing object
type ReconcileKnativeEventing struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
	config mf.Manifest
}

// Create manifestival resources and KnativeEventing, if necessary
func (r *ReconcileKnativeEventing) InjectClient(c client.Client) error {
	m, err := mf.NewManifest(*filename, *recursive, c)
	if err != nil {
		return err
	}
	r.config = m
	return r.ensureKnativeEventing()
}

// Reconcile reads that state of the cluster for a KnativeEventing object and makes changes based on the state read
// and what is in the KnativeEventing.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileKnativeEventing) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling KnativeEventing")

	// Fetch the KnativeEventing instance
	instance := &eventingv1alpha1.KnativeEventing{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			if isInteresting(request) {
				r.config.DeleteAll()
			}
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	if !isInteresting(request) {
		log.Info("Ignoring KnativeEventing", "namespace", instance.GetNamespace(), "name", instance.GetName())
		return reconcile.Result{}, r.ignore(instance)
	}

	// stages hook for future work (e.g. deleteObsoleteResources)
	stages := []func(*eventingv1alpha1.KnativeEventing) error{
		r.initStatus,
		r.install,
		r.checkDeployments,
	}

	for _, stage := range stages {
		if err := stage(instance); err != nil {
			return reconcile.Result{}, err
		}
	}

	return reconcile.Result{}, nil
}

// Initialize status conditions
func (r *ReconcileKnativeEventing) initStatus(instance *eventingv1alpha1.KnativeEventing) error {
	if len(instance.Status.Conditions) == 0 {
		instance.Status.InitializeConditions()
		if err := r.updateStatus(instance); err != nil {
			return err
		}
	}
	return nil
}

// Update the status subresource
func (r *ReconcileKnativeEventing) updateStatus(instance *eventingv1alpha1.KnativeEventing) error {

	// Account for https://github.com/kubernetes-sigs/controller-runtime/issues/406
	gvk := instance.GroupVersionKind()
	defer instance.SetGroupVersionKind(gvk)

	if err := r.client.Status().Update(context.TODO(), instance); err != nil {
		return err
	}
	return nil
}

// Apply the embedded resources
func (r *ReconcileKnativeEventing) install(instance *eventingv1alpha1.KnativeEventing) error {
	// Transform resources as appropriate
	fns := []mf.Transformer{
		mf.InjectOwner(instance),
		mf.InjectNamespace(instance.GetNamespace()),
		addSCCforSpecialClusterRoles,
	}
	r.config.Transform(fns...)

	if instance.Status.IsDeploying() {
		return nil
	}
	defer r.updateStatus(instance)

	// Apply the resources in the YAML file
	if err := r.config.ApplyAll(); err != nil {
		instance.Status.MarkInstallFailed(err.Error())
		return err
	}

	// Update status
	instance.Status.Version = version.Version
	log.Info("Install succeeded", "version", version.Version)
	instance.Status.MarkInstallSucceeded()
	return nil
}

// Check for all deployments available
// TODO: what about statefulsets?
func (r *ReconcileKnativeEventing) checkDeployments(instance *eventingv1alpha1.KnativeEventing) error {
	defer r.updateStatus(instance)
	available := func(d *appsv1.Deployment) bool {
		for _, c := range d.Status.Conditions {
			if c.Type == appsv1.DeploymentAvailable && c.Status == v1.ConditionTrue {
				return true
			}
		}
		return false
	}
	deployment := &appsv1.Deployment{}
	for _, u := range r.config.Resources {
		if u.GetKind() == "Deployment" {
			key := client.ObjectKey{Namespace: u.GetNamespace(), Name: u.GetName()}
			if err := r.client.Get(context.TODO(), key, deployment); err != nil {
				instance.Status.MarkDeploymentsNotReady()
				if errors.IsNotFound(err) {
					return nil
				}
				return err
			}
			if !available(deployment) {
				instance.Status.MarkDeploymentsNotReady()
				return nil
			}
		}
	}
	log.Info("All deployments are available")
	instance.Status.MarkDeploymentsAvailable()
	return nil
}

func addSCCforSpecialClusterRoles(u *unstructured.Unstructured) error {

	// these do need some openshift specific SCC
	clusterRoles := []string{
		"addressable-resolver",
		"broker-addressable-resolver",
		"channel-addressable-resolver",
		"eventing-broker-filter",
		"in-memory-channel-controller",
		"in-memory-channel-dispatcher",
		"knative-eventing-controller",
		"knative-eventing-webhook",
		"serving-addressable-resolver",
	}

	matchesClusterRole := func(cr string) bool {
		for _, i := range clusterRoles {
			if cr == i {
				return true
			}
		}
		return false
	}

	// massage the roles that require SCC on Openshift:
	if u.GetKind() == "ClusterRole" && matchesClusterRole(u.GetName()) {
		field, _, _ := unstructured.NestedFieldNoCopy(u.Object, "rules")
		// Required to properly run in OpenShift
		unstructured.SetNestedField(u.Object, append(field.([]interface{}), map[string]interface{}{
			"apiGroups":     []interface{}{"security.openshift.io"},
			"verbs":         []interface{}{"use"},
			"resources":     []interface{}{"securitycontextconstraints"},
			"resourceNames": []interface{}{"privileged", "anyuid"},
		}), "rules")
	}

	// fix the in-memory-channel-dispatcher to list configmaps
	// see https://github.com/knative/eventing/issues/1333 for more
	if u.GetKind() == "ClusterRole" && "in-memory-channel-dispatcher" == u.GetName() {
		field, _, _ := unstructured.NestedFieldNoCopy(u.Object, "rules")
		// Required to properly run in OpenShift
		unstructured.SetNestedField(u.Object, append(field.([]interface{}), map[string]interface{}{
			"apiGroups": []interface{}{""},
			"verbs":     []interface{}{"get", "list", "watch"},
			"resources": []interface{}{"configmaps"},
		}), "rules")
	}

	return nil
}

// Because it's effectively cluster-scoped, we only care about a
// single, named resource: knative-eventing/knative-eventing
func isInteresting(request reconcile.Request) bool {
	return request.Namespace == operand && request.Name == operand
}

// Reflect our ignorance in the KnativeEventing status
func (r *ReconcileKnativeEventing) ignore(instance *eventingv1alpha1.KnativeEventing) (err error) {
	err = r.initStatus(instance)
	if err == nil {
		msg := fmt.Sprintf("The only KnativeEventing resource that matters is %s/%s", operand, operand)
		instance.Status.MarkIgnored(msg)
		err = r.updateStatus(instance)
	}
	return
}

// If we can't find knative-eventing/knative-eventing, create it
func (r *ReconcileKnativeEventing) ensureKnativeEventing() (err error) {
	const path = "deploy/crds/eventing_v1alpha1_knativeeventing_cr.yaml"
	instance := &eventingv1alpha1.KnativeEventing{}
	key := client.ObjectKey{Namespace: operand, Name: operand}
	if err = r.client.Get(context.TODO(), key, instance); err != nil {
		var manifest mf.Manifest
		manifest, err = mf.NewManifest(path, false, r.client)
		if err == nil {
			// create namespace
			err = manifest.Apply(&r.config.Resources[0])
		}
		if err == nil {
			err = manifest.Transform(mf.InjectNamespace(operand))
		}
		if err == nil {
			err = manifest.ApplyAll()
		}
	}
	return
}
