package kabaneroplatform

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	kabanerov1alpha1 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Retrieves an OwnerRereference object populated with the Kabanero operator information.
func getOwnerReference(k *kabanerov1alpha1.Kabanero, c client.Client, reqLogger logr.Logger) (metav1.OwnerReference, error) {
	ownerIsController := true
	kInstance := &kabanerov1alpha1.Kabanero{}
	err := c.Get(context.Background(), types.NamespacedName{
		Name:      k.ObjectMeta.Name,
		Namespace: k.ObjectMeta.Namespace}, kInstance)

	if err != nil {
		return metav1.OwnerReference{}, err
	}

	ownerRef := metav1.OwnerReference{
		APIVersion: kInstance.TypeMeta.APIVersion,
		Kind:       kInstance.TypeMeta.Kind,
		Name:       kInstance.ObjectMeta.Name,
		UID:        kInstance.ObjectMeta.UID,
		Controller: &ownerIsController,
	}

	reqLogger.Info(fmt.Sprintf("getOwnerReference: OwnerReference: %v", ownerRef))

	return ownerRef, err
}

// Retrieves the deployment's "Available" status condition.
func getDeploymentStatus(c client.Client, name string, namespace string) (bool, error) {
	// Check if the Deployment resource exists.
	dInstance := &appsv1.Deployment{}
	err := c.Get(context.Background(), types.NamespacedName{
		Name:      name,
		Namespace: namespace}, dInstance)

	if err != nil {
		return false, err
	}

	// Retrieve the status condition.
	for _, condition := range dInstance.Status.Conditions {
		if condition.Type == appsv1.DeploymentAvailable {
			if condition.Status == corev1.ConditionTrue {
				return true, nil
			} else {
				return false, fmt.Errorf("Deployment Available status condition was %v", condition.Status)
			}
		}
	}

	// Did not find the condition
	return false, fmt.Errorf("Deployment did not contains an Available status condition")
}
