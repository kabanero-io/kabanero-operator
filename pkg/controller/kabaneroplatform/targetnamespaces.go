package kabaneroplatform
import (
	"context"

	kabanerov1alpha2 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha2"
	
	"github.com/go-logr/logr"
	
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func reconcileTargetNamespaces(ctx context.Context, k *kabanerov1alpha2.Kabanero, cl client.Client, reqLogger logr.Logger) error {

	// Rolebinding Template
	ownerIsController := true
	rolebindingResource := rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kabanero-pipeline-deploy-rolebinding",
			OwnerReferences: []metav1.OwnerReference{
				metav1.OwnerReference{
					APIVersion: k.TypeMeta.APIVersion,
					Kind:       k.TypeMeta.Kind,
					Name:       k.ObjectMeta.Name,
					UID:        k.ObjectMeta.UID,
					Controller: &ownerIsController,
				},
			},
		},
		Subjects: []rbacv1.Subject{
			rbacv1.Subject{
				Kind: "User",
				Name: "kabanero-pipeline",
				APIGroup: "rbac.authorization.k8s.io",
			},
		},
		RoleRef: rbacv1.RoleRef{
			Kind: "ClusterRole",
			Name: "kabanero-pipeline-deploy-role",
			APIGroup: "rbac.authorization.k8s.io",
		},
	}


	// List of Namespaces we want to bind
	targetnamespaceList := k.Spec.TargetNamespaces
	
	// If targetNamespaces is empty, default to binding to kabanero
	if len(targetnamespaceList) == 0 {
		targetnamespaceList = append(targetnamespaceList, "kabanero")
	}

	/* Get all Rolebindings named kabanero-pipeline-deploy-rolebinding
		Structured method only lists RoleBindings in kabanero namespace. Maybe due to client scoping?
		
	rolebindingList := &rbacv1.RoleBindingList{}
	err := cl.List(ctx, rolebindingList, client.MatchingFields{"metadata.name": "kabanero-pipeline-deploy-rolebinding"})
	if err != nil {
		return err
	}
	*/

	// Get all Rolebindings named kabanero-pipeline-deploy-rolebinding
	rolebindingList := &unstructured.UnstructuredList{}
	rolebindingList.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "rbac.authorization.k8s.io",
		Kind:    "RoleBindingList",
		Version: "v1",
	})
	err := cl.List(ctx, rolebindingList, client.MatchingFields{"metadata.name": "kabanero-pipeline-deploy-rolebinding"})
	if err != nil {
		return err
	}

	// For each Rolebinding
	for _, rolebinding := range rolebindingList.Items {
		matchFound := false
		// If the Rolebinding namespace is in the targetNamespace list
		for i, targetnamespace := range targetnamespaceList {
			if rolebinding.GetNamespace() == targetnamespace {
			
				// Fill in the namespace for the template
				desiredRolebinding := rolebindingResource
				desiredRolebinding.ObjectMeta.Namespace = targetnamespace
			
				// Check if the existing Rolebinding matches the desired template, and skip?
				// Update may handle this already
				
				// Apply the rolebinding
				cl.Update(ctx, &desiredRolebinding)
				
				// Remove the namespace from the list of Namespaces remaining to bind
				targetnamespaceList[i] = targetnamespaceList[len(targetnamespaceList)-1] // Copy last element to index i.
				targetnamespaceList[len(targetnamespaceList)-1] = ""                     // Erase last element (write zero value).
				targetnamespaceList = targetnamespaceList[:len(targetnamespaceList)-1]   // Truncate slice.

				matchFound = true
				break
			}
		}

		// If the Rolebinding does not match a targetNamespace, delete
		if matchFound == false {
			cl.Delete(ctx, &rolebinding)
		}
	}
	
	// For remaining namespaces in the list, Create the Rolebinding
	for _, targetnamespace := range targetnamespaceList {
		// Check if the namespace exists
		namespace := &unstructured.Unstructured{}
		namespace.SetGroupVersionKind(schema.GroupVersionKind{
				Group:   "",
				Kind:    "Namespace",
				Version: "v1",
		})
		err = cl.Get(ctx, client.ObjectKey{Namespace: targetnamespace, Name: targetnamespace,}, namespace)
		if err == nil {
			// Fill in the namespace for the template and Create
			desiredRolebinding := rolebindingResource
			desiredRolebinding.ObjectMeta.Namespace = targetnamespace
			cl.Create(ctx, &desiredRolebinding)
		}
	}
	
	return nil
}
