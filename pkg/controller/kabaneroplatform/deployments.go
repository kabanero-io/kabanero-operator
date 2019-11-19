package kabaneroplatform

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	kabanerov1alpha1 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
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

// Creates and deploys a Deployment resource with the following assumptions:
// 1) The name, app name, service account name are all the same
// 2) Exposes port 9443 (SSL)
// 3) Image pull policy is "Always"
// 4) Single replica
// 5) Owner is a Kabanero instance (Supplied)
func createDeployment(k *kabanerov1alpha1.Kabanero, clientset *kubernetes.Clientset, c client.Client, name string, image string, env []corev1.EnvVar, envFrom []corev1.EnvFromSource, livenessProbe *corev1.Probe, reqLogger logr.Logger) error {
	cl := clientset.AppsV1().Deployments(k.ObjectMeta.Namespace)

	// Check if the Deployment resource already exists.
	dInstance := &appsv1.Deployment{}
	err := c.Get(context.Background(), types.NamespacedName{
		Name:      name,
		Namespace: k.ObjectMeta.Namespace}, dInstance)

	deploymentExists := true
	if err != nil {
		if apierrors.IsNotFound(err) == false {
			return err
		}

		// The deployment does not already exist.  Create one.
		deploymentExists = false

		// Gather Kabanero operator ownerReference information.
		ownerRef, err := getOwnerReference(k, c, reqLogger)
		if err != nil {
			return err
		}

		// Initialize the deployment
		var repCount int32 = 1
		dInstance = &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: ownerRef.APIVersion,
						Kind:       ownerRef.Kind,
						Name:       ownerRef.Name,
						UID:        ownerRef.UID,
						Controller: ownerRef.Controller,
					},
				},
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: &repCount,
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app": name,
					},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"app": name,
						},
					},
					Spec: corev1.PodSpec{
						ServiceAccountName: name,
						Containers: []corev1.Container{
							{
								Name:            name,
								ImagePullPolicy: "Always",
								Ports: []corev1.ContainerPort{
									{
										ContainerPort: 9443,
									},
								},
							},
						},
					},
				},
			},
		}
	}

	// Here we update the things that can change.  In the future we could
	// consider re-applying all the fields in case someone hand-edited the
	// deployment object in an incompatible way.
	dInstance.Spec.Template.Spec.Containers[0].Env = env
	dInstance.Spec.Template.Spec.Containers[0].EnvFrom = envFrom
	dInstance.Spec.Template.Spec.Containers[0].Image = image
	dInstance.Spec.Template.Spec.Containers[0].LivenessProbe = livenessProbe

	if deploymentExists == false {
		reqLogger.Info(fmt.Sprintf("createDeployment: Deployment for create: %v", dInstance))

		_, err = cl.Create(dInstance)
	} else {
		reqLogger.Info(fmt.Sprintf("createDeployment: Deployment for update: %v", dInstance))

		_, err = cl.Update(dInstance)
	}

	return err
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
