package kabaneroplatform

import (
	"context"
	"errors"
	"fmt"

	kabanerov1alpha2 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha2"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"testing"
)

func init() {
	logf.SetLogger(testLogger{})
}

var nslog = logf.Log.WithName("targetnamespaces_test")

// Unit test Kube client
type targetnamespaceTestClient struct {
	// Role bindings that the client knows about.
	objs map[client.ObjectKey]bool

	// Namespaces that the client knows about
	namespaces map[string]bool
}

func (c targetnamespaceTestClient) Get(ctx context.Context, key client.ObjectKey, obj runtime.Object) error {
	fmt.Printf("Received Get() for %v\n", key.Name)
	u, ok := obj.(*unstructured.Unstructured)
	if !ok {
		fmt.Printf("Received invalid target object for get: %v\n", obj)
		return errors.New("Get only supports setting into Unstructured")
	}
	_, ok = c.namespaces[key.Name]
	if !ok {
		return apierrors.NewNotFound(schema.GroupResource{}, key.Name)
	}
	u.SetName(key.Name)
	u.SetNamespace(key.Namespace)
	return nil
}
func (c targetnamespaceTestClient) List(ctx context.Context, list runtime.Object, opts ...client.ListOption) error {
	return nil
}
func (c targetnamespaceTestClient) Create(ctx context.Context, obj runtime.Object, opts ...client.CreateOption) error {
	binding, ok := obj.(*rbacv1.RoleBinding)
	if !ok {
		fmt.Printf("Received invalid create: %v\n", obj)
		return errors.New("Create only supports RoleBinding")
	}

	fmt.Printf("Received Create() for %v\n", binding.GetName())
	key := client.ObjectKey{Name: binding.GetName(), Namespace: binding.GetNamespace()}
	_, ok = c.objs[key]
	if ok {
		fmt.Printf("Receive create object already exists: %v/%v\n", binding.GetNamespace(), binding.GetName())
		return apierrors.NewAlreadyExists(schema.GroupResource{}, binding.GetName())
	}

	c.objs[key] = true
	return nil
}
func (c targetnamespaceTestClient) Delete(ctx context.Context, obj runtime.Object, opts ...client.DeleteOption) error {
	binding, ok := obj.(*rbacv1.RoleBinding)
	if !ok {
		fmt.Printf("Received invalid delete: %v\n", obj)
		return errors.New("Delete only supports RoleBinding")
	}

	fmt.Printf("Received Delete() for %v\n", binding.GetName())
	key := client.ObjectKey{Name: binding.GetName(), Namespace: binding.GetNamespace()}
	_, ok = c.objs[key]
	if !ok {
		fmt.Printf("Received delete for an object that does not exist: %v\n", obj)
		return apierrors.NewNotFound(schema.GroupResource{}, binding.GetName())
	}
	delete(c.objs, key)
	return nil
}
func (c targetnamespaceTestClient) DeleteAllOf(ctx context.Context, obj runtime.Object, opts ...client.DeleteAllOfOption) error {
	return errors.New("DeleteAllOf is not supported")
}
func (c targetnamespaceTestClient) Update(ctx context.Context, obj runtime.Object, opts ...client.UpdateOption) error {
	binding, ok := obj.(*rbacv1.RoleBinding)
	if !ok {
		fmt.Printf("Received invalid update: %v\n", obj)
		return errors.New("Update only supports RoleBinding")
	}

	fmt.Printf("Received Update() for %v\n", binding.GetName())
	key := client.ObjectKey{Name: binding.GetName(), Namespace: binding.GetNamespace()}
	_, ok = c.objs[key]
	if !ok {
		fmt.Printf("Received update for object that does not exist: %v\n", obj)
		return apierrors.NewNotFound(schema.GroupResource{}, binding.GetName())
	}
	return nil
}
func (c targetnamespaceTestClient) Status() client.StatusWriter { return c }

func (c targetnamespaceTestClient) Patch(ctx context.Context, obj runtime.Object, patch client.Patch, opts ...client.PatchOption) error {
	return errors.New("Patch is not supported")
}

// Apply the role bindings to an existing namespace
func TestReconcileTargetNamespaces(t *testing.T) {
	targetNamespace := "fred"
	k := kabanerov1alpha2.Kabanero{
		ObjectMeta: metav1.ObjectMeta{Name: "kabanero", Namespace: "kabanero"},
		Spec: kabanerov1alpha2.KabaneroSpec{
			TargetNamespaces: []string{targetNamespace},
		},
	}

	existingNamespaces := make(map[string]bool)
	existingNamespaces[targetNamespace] = true
	client := targetnamespaceTestClient{map[client.ObjectKey]bool{}, existingNamespaces}
	
	err := reconcileTargetNamespaces(context.TODO(), &k, client, nslog)

	if err != nil {
		t.Fatal("Returned error: " + err.Error())
	}

	// Make sure the kabanero status was updated with the target namespace
	if len(k.Status.TargetNamespaces.Namespaces) != 1 {
		t.Fatal(fmt.Sprintf("Kabanero status should have 1 target namespace, but has %v: %v", len(k.Status.TargetNamespaces.Namespaces), k.Status.TargetNamespaces.Namespaces))
	}

	if k.Status.TargetNamespaces.Namespaces[0] != targetNamespace {
		t.Fatal(fmt.Sprintf("Kabanero status target namespace should be %v, but is %v", targetNamespace, k.Status.TargetNamespaces.Namespaces[0]))
	}

	if k.Status.TargetNamespaces.Ready != "True" {
		t.Fatal(fmt.Sprintf("Kabanero target namespace status is not True: %v", k.Status.TargetNamespaces.Ready))
	}

	if len(k.Status.TargetNamespaces.Message) != 0 {
		t.Fatal(fmt.Sprintf("Kabanero target namespace status contains an error message: %v", k.Status.TargetNamespaces.Message))
	}

	// Make sure the RoleBindings got added in the correct namespace.
	if len(client.objs) != 2 {
		t.Fatal(fmt.Sprintf("Should have created two RoleBindings, but created %v: %#v", len(client.objs), client.objs))
	}

	for key, _ := range client.objs {
		if key.Namespace != targetNamespace {
			t.Fatal(fmt.Sprintf("Should have created RoleBinding in %v namespace, but created in %v namespace", targetNamespace, key.Namespace))
		}
	}
}

// Apply the role bindings to a namespace that does not exist.
func TestReconcileTargetNamespacesNamespaceNotExist(t *testing.T) {
	targetNamespace := "fred"
	activeNamespace := "kabanero"
	k := kabanerov1alpha2.Kabanero{
		ObjectMeta: metav1.ObjectMeta{Name: "kabanero", Namespace: "kabanero"},
		Spec: kabanerov1alpha2.KabaneroSpec{
			TargetNamespaces: []string{targetNamespace},
		},
		Status: kabanerov1alpha2.KabaneroStatus {
			TargetNamespaces: kabanerov1alpha2.TargetNamespaceStatus {
				Namespaces: []string{activeNamespace},
				Ready: "True",
			},
		},
	}

	// Set up pre-existing objects
	existingNamespaces := make(map[string]bool)
	existingNamespaces[activeNamespace] = true
	existingRoleBinding := client.ObjectKey{Name: "kabanero-pipeline-deploy-rolebinding", Namespace: activeNamespace}
	existingRoleBindings := make(map[client.ObjectKey]bool)
	existingRoleBindings[existingRoleBinding] = true
	client := targetnamespaceTestClient{existingRoleBindings, existingNamespaces}
	
	err := reconcileTargetNamespaces(context.TODO(), &k, client, nslog)

	if err == nil {
		t.Fatal("Did not return an error, but should have because namespace does not exist")
	}

	// Make sure the kabanero status was not updated with the target namespace,
	// since it did not exist.
	if len(k.Status.TargetNamespaces.Namespaces) != 0 {
		t.Fatal(fmt.Sprintf("Kabanero status should have 0 target namespace, but has %v: %v", len(k.Status.TargetNamespaces.Namespaces), k.Status.TargetNamespaces.Namespaces))
	}

	if k.Status.TargetNamespaces.Ready != "False" {
		t.Fatal(fmt.Sprintf("Kabanero target namespace status is not False: %v", k.Status.TargetNamespaces.Ready))
	}

	if len(k.Status.TargetNamespaces.Message) == 0 {
		t.Fatal("Kabanero target namespace status contains no error message")
	}

	// Make sure the RoleBinding map got cleared.
	if len(client.objs) != 0 {
		t.Fatal(fmt.Sprintf("Should have created 0 RoleBindings, but created %v: %#v", len(client.objs), client.objs))
	}

	// OK, now create the namespace and make sure things resolve as per normal.
	existingNamespaces[targetNamespace] = true

	err = reconcileTargetNamespaces(context.TODO(), &k, client, nslog)

	if err != nil {
		t.Fatal("Returned error: " + err.Error())
	}

	// Make sure the kabanero status was updated with the target namespace
	if len(k.Status.TargetNamespaces.Namespaces) != 1 {
		t.Fatal(fmt.Sprintf("Kabanero status should have 1 target namespace, but has %v: %v", len(k.Status.TargetNamespaces.Namespaces), k.Status.TargetNamespaces.Namespaces))
	}

	if k.Status.TargetNamespaces.Namespaces[0] != targetNamespace {
		t.Fatal(fmt.Sprintf("After NS create, Kabanero status target namespace should be %v, but is %v", targetNamespace, k.Status.TargetNamespaces.Namespaces[0]))
	}

	if k.Status.TargetNamespaces.Ready != "True" {
		t.Fatal(fmt.Sprintf("After NS create, Kabanero target namespace status is not True: %v", k.Status.TargetNamespaces.Ready))
	}

	if len(k.Status.TargetNamespaces.Message) != 0 {
		t.Fatal(fmt.Sprintf("After NS create, Kabanero target namespace status contains an error message: %v", k.Status.TargetNamespaces.Message))
	}

	// Make sure the RoleBindings got added in the correct namespace.
	if len(client.objs) != 2 {
		t.Fatal(fmt.Sprintf("After NS create, should have created two RoleBindings, but created %v: %#v", len(client.objs), client.objs))
	}

	for key, _ := range client.objs {
		if key.Namespace != targetNamespace {
			t.Fatal(fmt.Sprintf("After NS create, should have created RoleBinding in %v namespace, but created in %v namespace", targetNamespace, key.Namespace))
		}
	}
}

// Apply the role bindings to a namespace that does not exist.
func TestTargetNamespacesGotDeleted(t *testing.T) {
	targetNamespace1 := "fred"
	targetNamespace2 := "george"
	activeNamespace1 := "fred"
	activeNamespace2 := "george"

	k := kabanerov1alpha2.Kabanero{
		ObjectMeta: metav1.ObjectMeta{Name: "kabanero", Namespace: "kabanero"},
		Spec: kabanerov1alpha2.KabaneroSpec{
			TargetNamespaces: []string{targetNamespace1, targetNamespace2},
		},
		Status: kabanerov1alpha2.KabaneroStatus {
			TargetNamespaces: kabanerov1alpha2.TargetNamespaceStatus {
				Namespaces: []string{activeNamespace1, activeNamespace2},
				Ready: "True",
			},
		},
	}

	// Set up pre-existing objects
	existingNamespaces := make(map[string]bool)
	existingNamespaces[activeNamespace1] = true
	existingRoleBinding1 := client.ObjectKey{Name: "kabanero-pipeline-deploy-rolebinding", Namespace: activeNamespace1}
	existingRoleBinding2 := client.ObjectKey{Name: "kabanero-cli-deploy-rolebinding", Namespace: activeNamespace1}
	
	existingRoleBindings := make(map[client.ObjectKey]bool)
	existingRoleBindings[existingRoleBinding1] = true
	existingRoleBindings[existingRoleBinding2] = true
	client := targetnamespaceTestClient{existingRoleBindings, existingNamespaces}
	
	err := reconcileTargetNamespaces(context.TODO(), &k, client, nslog)

	if err == nil {
		t.Fatal("Did not return an error, but should have because namespace \"george\" does not exist")
	}

	// Make sure the kabanero status was not updated with the target namespace,
	// since it did not exist.
	if len(k.Status.TargetNamespaces.Namespaces) != 1 {
		t.Fatal(fmt.Sprintf("Kabanero status should have 1 target namespace, but has %v: %v", len(k.Status.TargetNamespaces.Namespaces), k.Status.TargetNamespaces.Namespaces))
	}

	if k.Status.TargetNamespaces.Namespaces[0] != "fred" {
		t.Fatal(fmt.Sprintf("Kabanero status target namespace is not \"fred\", but is %v", k.Status.TargetNamespaces.Namespaces[0]))
	}
	
	if k.Status.TargetNamespaces.Ready != "False" {
		t.Fatal(fmt.Sprintf("Kabanero target namespace status is not False: %v", k.Status.TargetNamespaces.Ready))
	}

	if len(k.Status.TargetNamespaces.Message) == 0 {
		t.Fatal("Kabanero target namespace status contains no error message")
	}

	// Make sure the RoleBinding map got cleared.
	if len(client.objs) != 2 {
		t.Fatal(fmt.Sprintf("Should have created two RoleBindings, but created %v: %#v", len(client.objs), client.objs))
	}
}

// Test callout from finalizer
func TestCleanupTargetNamespaces(t *testing.T) {
	targetNamespace := "fred"
	k := kabanerov1alpha2.Kabanero{
		ObjectMeta: metav1.ObjectMeta{Name: "kabanero", Namespace: "kabanero"},
		Spec: kabanerov1alpha2.KabaneroSpec{
			TargetNamespaces: []string{targetNamespace},
		},
		Status: kabanerov1alpha2.KabaneroStatus {
			TargetNamespaces: kabanerov1alpha2.TargetNamespaceStatus {
				Namespaces: []string{targetNamespace},
				Ready: "True",
			},
		},
	}

	// Set up pre-existing objects
	existingNamespaces := make(map[string]bool)
	existingNamespaces[targetNamespace] = true
	existingRoleBinding1 := client.ObjectKey{Name: "kabanero-pipeline-deploy-rolebinding", Namespace: targetNamespace}
	existingRoleBinding2 := client.ObjectKey{Name: "kabanero-cli-deploy-rolebinding", Namespace: targetNamespace}
	existingRoleBindings := make(map[client.ObjectKey]bool)
	existingRoleBindings[existingRoleBinding1] = true
	existingRoleBindings[existingRoleBinding2] = true
	client := targetnamespaceTestClient{existingRoleBindings, existingNamespaces}
	
	err := cleanupTargetNamespaces(context.TODO(), &k, client)

	if err != nil {
		t.Fatal("Returned error: " + err.Error())
	}

	if len(existingRoleBindings) != 0 {
		t.Fatal(fmt.Sprintf("There were %v bindings left in the map after cleanup: %#v", len(existingRoleBindings), existingRoleBindings))
	}
}
