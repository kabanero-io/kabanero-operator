package kabaneroplatform

import (
	"context"
	"errors"
	"fmt"
	"strings"

	kabanerov1alpha2 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha2"

	"github.com/go-logr/logr"

	rbacv1 "k8s.io/api/rbac/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type targetNamespaceRoleBindingTemplate struct {
	name            string
	saName          string
	saNamespace     string
	clusterRoleName string
}

func (info targetNamespaceRoleBindingTemplate) generate(targetNamespace string) rbacv1.RoleBinding {
	return rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      info.name,
			Namespace: targetNamespace,
		},
		Subjects: []rbacv1.Subject{
			rbacv1.Subject{
				Kind:      "ServiceAccount",
				Name:      info.saName,
				Namespace: info.saNamespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			Kind:     "ClusterRole",
			Name:     info.clusterRoleName,
			APIGroup: "rbac.authorization.k8s.io",
		},
	}
}

// We're going to target the current namespace, and the list of target
// namespaces from the Kabanero CR instance.
func getTargetNamespaces(targetNamespaces []string, defaultNamespace string) []string {
	targetnamespaceList := targetNamespaces

	// If targetNamespaces is empty, default to binding to kabanero
	if len(targetnamespaceList) == 0 {
		targetnamespaceList = append(targetnamespaceList, defaultNamespace)
	}

	return targetnamespaceList
}

// Create the binding templates
func createBindingTemplates(saNamespace string) []targetNamespaceRoleBindingTemplate {
	return []targetNamespaceRoleBindingTemplate{
		{
			name:            "kabanero-pipeline-deploy-rolebinding",
			saName:          "kabanero-pipeline",
			saNamespace:     saNamespace,
			clusterRoleName: "kabanero-pipeline-deploy-role",
		},
		{
			name:            "kabanero-cli-deploy-rolebinding",
			saName:          "kabanero-cli",
			saNamespace:     saNamespace,
			clusterRoleName: "kabanero-cli-service-deployments-role",
		},
	}
}

func reconcileTargetNamespaces(ctx context.Context, k *kabanerov1alpha2.Kabanero, cl client.Client, reqLogger logr.Logger) error {

	// Owner reference for same-namespace bindings
	ownerIsController := true
	ownerReference := metav1.OwnerReference{
		APIVersion: k.TypeMeta.APIVersion,
		Kind:       k.TypeMeta.Kind,
		Name:       k.ObjectMeta.Name,
		UID:        k.ObjectMeta.UID,
		Controller: &ownerIsController,
	}

	// Be sure each requested namespace exists.  This will catch namespaces added to the list, as well as
	// namespaces that were deleted but not removed from the targetNamespaces list.
	specTargetNamespaces := sets.NewString(getTargetNamespaces(k.Spec.TargetNamespaces, k.GetNamespace())...)
	var errorNamespaces []string
	for namespace, _ := range specTargetNamespaces {
		exists, err := namespaceExists(ctx, namespace, cl)
		if err != nil {
			reqLogger.Error(err, fmt.Sprintf("Could not check status of namespace %v", namespace))
			errorNamespaces = append(errorNamespaces, namespace)
		}
		if exists == false {
			reqLogger.Error(nil, fmt.Sprintf("Target namespace %v does not exist", namespace))
			errorNamespaces = append(errorNamespaces, namespace)
		}
	}

	for _, namespace := range errorNamespaces {
		delete(specTargetNamespaces, namespace)
	}

	// TODO: did I do this right?  need to process the namespaces, then look at errorNamespaces and
	//       generate an error message for namespaces that did not exist.  Once we have a watch set
	//       up, that should take care of partially active lists, and the delete case.

	// Compute the new, deleted, and common namespace names
	statusTargetNamespaces := sets.NewString(getTargetNamespaces(k.Status.TargetNamespaces.Namespaces, k.GetNamespace())...)
	oldNamespaces := statusTargetNamespaces.Difference(specTargetNamespaces)
	newNamespaces := specTargetNamespaces.Difference(statusTargetNamespaces)
	unchangedNamespaces := specTargetNamespaces.Intersection(statusTargetNamespaces)

	// Create the templates
	bindingTemplates := createBindingTemplates(k.GetNamespace())

	// For removed namespaces, delete the role bindings
	for namespace, _ := range oldNamespaces {
		for _, bindingTemplate := range bindingTemplates {
			template := bindingTemplate.generate(namespace)
			reqLogger.Info(fmt.Sprintf("Deleting RoleBinding %v for removed target namespace %v", template.GetName(), template.GetNamespace()))
			cl.Delete(ctx, &template)
		}
	}

	// For new namespaces, create the role bindings
	for namespace, _ := range newNamespaces {
		for _, bindingTemplate := range bindingTemplates {
			template := bindingTemplate.generate(namespace)
			if k.GetNamespace() == namespace {
				template.ObjectMeta.OwnerReferences = []metav1.OwnerReference{ownerReference}
			}
			reqLogger.Info(fmt.Sprintf("Creating RoleBinding %v for added target namespace %v", template.GetName(), template.GetNamespace()))
			cl.Create(ctx, &template)
		}
	}

	// For unchanged namespaces, validate the role bindings
	for namespace, _ := range unchangedNamespaces {
		for _, bindingTemplate := range bindingTemplates {
			template := bindingTemplate.generate(namespace)
			if k.GetNamespace() == namespace {
				template.ObjectMeta.OwnerReferences = []metav1.OwnerReference{ownerReference}
			}
			reqLogger.Info(fmt.Sprintf("Updating RoleBinding %v for unchanged target namespace %v", template.GetName(), template.GetNamespace()))
			cl.Update(ctx, &template)
		}
	}

	// Update the Status to reflect the new target namespaces.
	k.Status.TargetNamespaces.Namespaces = nil
	for _, namespace := range k.Spec.TargetNamespaces {
		isErrorNamespace := false
		for _, errorNamespace := range errorNamespaces {
			if errorNamespace == namespace {
				isErrorNamespace = true
				break
			}
		}
		if isErrorNamespace == false {
			k.Status.TargetNamespaces.Namespaces = append(k.Status.TargetNamespaces.Namespaces, namespace)
		}
	}

	if len(errorNamespaces) == 0 {
		k.Status.TargetNamespaces.Ready = "True"
		k.Status.TargetNamespaces.Message = ""
	} else {
		k.Status.TargetNamespaces.Ready = "False"
		k.Status.TargetNamespaces.Message = fmt.Sprintf("The following namespaces could not be processed: %v", strings.Join(errorNamespaces, ","))
		return errors.New(k.Status.TargetNamespaces.Message)
	}

	return nil
}

// Checks if a namespace exists.  If an unknown error occurs, return that too.
func namespaceExists(ctx context.Context, inNamespace string, cl client.Client) (bool, error) {
	namespace := &unstructured.Unstructured{}
	namespace.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "",
		Kind:    "Namespace",
		Version: "v1",
	})
	err := cl.Get(ctx, client.ObjectKey{Namespace: inNamespace, Name: inNamespace}, namespace)
	if err == nil {
		return true, nil
	}

	if kerrors.IsNotFound(err) {
		return false, nil
	}

	return false, err
}

// Returns the readiness status of the target namespaces.  Presently the status
// is determined as the namespaces are activated.  We are just reporting that
// status here.
func getTargetNamespacesStatus(k *kabanerov1alpha2.Kabanero) (bool, error) {
	return k.Status.TargetNamespaces.Ready == "True", nil
}

// Clean up the cross-namespace bindings that we created (deleting the
// Kabanero CR instance won't delete these because cross-namespace owner
// references are not allowed by Kubernetes).
func cleanupTargetNamespaces(ctx context.Context, k *kabanerov1alpha2.Kabanero, cl client.Client) error {
	// Create the templates
	bindingTemplates := createBindingTemplates(k.GetNamespace())

	for _, namespace := range getTargetNamespaces(k.Status.TargetNamespaces.Namespaces, k.GetNamespace()) {
		for _, bindingTemplate := range bindingTemplates {
			template := bindingTemplate.generate(namespace)
			cl.Delete(ctx, &template)
		}
	}

	return nil
}
