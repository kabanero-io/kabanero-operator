package kabaneroplatform

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-logr/logr"
	mf "github.com/jcrossley3/manifestival"
	kabanerov1alpha1 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha1"
	kutils "github.com/kabanero-io/kabanero-operator/pkg/controller/kabaneroplatform/utils"
	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
	rlog "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var kanlog = rlog.Log.WithName("kabanero-kappnav")

const currentVersion = "0.1.0"
const resourcesFileName = "config/orchestrations/kappnav/0.1/kappnav-0.1.0.yaml"
const crFileName = "config/orchestrations/kappnav/0.1/kappnav-cr-0.1.0.yaml"

// Reconciles the Kubernetes Application Navigator.
func reconcileKappnav(ctx context.Context, k *kabanerov1alpha1.Kabanero, c client.Client) error {
	logger := kanlog.WithValues("Kabanero instance namespace", k.Namespace, "Kabanero instance Name", k.Name)
	logger.Info(fmt.Sprintf("reconcileKappnav: Reconciling KAppNav. Enabled: %v", k.Spec.Kappnav.Enable))

	// Enable path. Apply the needed resource yamls.
	if k.Spec.Kappnav.Enable {
		err := applyResources(ctx, k, c, resourcesFileName, logger)
		if err != nil {
			return err
		}

		crdActive, err := isKNavAppCRDActive(logger)
		if err != nil {
			return err
		}

		if crdActive {
			err = applyResources(ctx, k, c, crFileName, logger)
		}

		return err
	}

	// Disable path. Do basic cleanup for now to be consistent with kabanero instance cleanup.
	err := cleanupKappnav(ctx, k, c)

	return err
}

// Applies resource files.
func applyResources(ctx context.Context, k *kabanerov1alpha1.Kabanero, c client.Client, filename string, logger logr.Logger) error {
	m, err := mf.NewManifest(filename, true, c)
	if err != nil {
		return err
	}

	transforms := []mf.Transformer{
		mf.InjectOwner(k),
		mf.InjectNamespace(k.GetNamespace()),
	}

	err = m.Transform(transforms...)
	if err != nil {
		return err
	}

	logger.Info(fmt.Sprintf("applyResources: KAppNav Resources: %v", m))
	err = m.ApplyAll()

	return err
}

// Returns true if the KAppNav CRD is active. False, otherwise.
func isKNavAppCRDActive(logger logr.Logger) (bool, error) {
	config, err := clientcmd.BuildConfigFromFlags("", "")
	if err != nil {
		logger.Error(err, fmt.Sprintf("isKNavAppCRDEstablished: Unable to get config object."))
		return false, err
	}

	kExtClientset, err := clientset.NewForConfig(config)
	if err != nil {
		logger.Error(err, fmt.Sprintf("isKNavAppCRDEstablished: Unable to get the api extensions client."))
		return false, err
	}

	err = kutils.Retry(12, 5*time.Second, func() (bool, error) {
		active := false
		crd, err := kExtClientset.ApiextensionsV1beta1().CustomResourceDefinitions().Get("kappnavs.charts.helm.k8s.io", metav1.GetOptions{})
		if err != nil {
			if apiErrors.IsNotFound(err) {
				return active, nil
			}

			return active, err
		}

		// We found the CRD object. Check that it is active.
		for _, condition := range crd.Status.Conditions {
			if condition.Type == apiextv1beta1.Established {
				if condition.Status == apiextv1beta1.ConditionTrue {
					active = true
					break
				}
			}
		}

		return active, nil
	})

	if err != nil {
		return false, err
	}

	return true, err
}

// Retrieves the Kubernetes Application Navigator deployment status.
func getKappnavStatus(k *kabanerov1alpha1.Kabanero, c client.Client) (bool, error) {
	// If disabled. Nothing to do.
	if !k.Spec.Kappnav.Enable {
		k.Status.Kappnav = nil
		return true, nil
	}

	// KNavApp is optional. The status was defined as a pointer so that it is not displayed
	// in the kabanero instance data if KAppNav is disabled. That is because structures are
	// never 'empty' for json tagging 'omitempty' to take effect.
	// We need to create the structure here before we use it.
	kstatus := kabanerov1alpha1.KappnavStatus{}
	k.Status.Kappnav = &kstatus
	k.Status.Kappnav.Version = currentVersion
	k.Status.Kappnav.ErrorMessage = ""
	k.Status.Kappnav.Ready = "False"

	clientset, err := getClientset()
	if err != nil {
		k.Status.Kappnav.ErrorMessage = "Failed to obtain client deployments interface."
		return false, err
	}

	// Check if the deployment is active.
	deploymentClient := clientset.AppsV1().Deployments(k.GetNamespace())
	deployment, err := deploymentClient.Get("helm-operator", metav1.GetOptions{})

	if err != nil {
		message := "Unable to retrieve the kappnav helm-operator deployment object."
		kanlog.Error(err, message)
		k.Status.Kappnav.ErrorMessage = message + ": " + err.Error()
		return false, err
	}

	conditions := deployment.Status.Conditions
	ready := false
	for _, condition := range conditions {
		if strings.ToLower(string(condition.Type)) == "available" {
			if strings.ToLower(string(condition.Status)) == "true" {
				ready = true
			} else {
				k.Status.Kappnav.ErrorMessage = condition.Message
			}

			break
		}
	}

	if !ready {
		return false, err
	}

	// Check if a pod serving the KappNav UI is active.
	options := metav1.ListOptions{LabelSelector: "app.kubernetes.io/component=kappnav-ui"}
	pods, err := clientset.CoreV1().Pods(k.ObjectMeta.Namespace).List(options)

	if err != nil {
		message := "Pods deployments with label app.kubernetes.io/component=kappnav-ui under the namespace of " + k.ObjectMeta.Namespace + " could not be retrieved."
		kanlog.Error(err, message)
		k.Status.Kappnav.ErrorMessage = message + ": " + err.Error()
		return false, err
	}

	finalErrorMessage := ""
	if pods != nil && len(pods.Items) == 0 {
		finalErrorMessage = "Pod deployments with label app.kubernetes.io/component=kappnav-ui under the namespace of " + k.ObjectMeta.Namespace + " were not found."
	}

	ready = false
	for _, pod := range pods.Items {
		for _, condition := range pod.Status.Conditions {
			if strings.ToLower(string(condition.Type)) == "ready" {
				readyStatus := string(condition.Status)
				if strings.ToLower(readyStatus) == "true" {
					ready = true
				} else {
					finalErrorMessage += "Pod " + pod.Name + " not ready: " + condition.Message + ". "
				}
				break
			}
		}
		if ready {
			break
		}
	}

	if ready {
		k.Status.Kappnav.Ready = "True"
	} else {
		k.Status.Kappnav.ErrorMessage = finalErrorMessage
	}

	return ready, err
}

// Performs cleanup processing.
func cleanupKappnav(ctx context.Context, k *kabanerov1alpha1.Kabanero, c client.Client) error {
	// delete the CR instance.
	// If we could not delete the instance, let's not make things worse
	// with the kappnav finalizer by deleting the rest of the resources.
	err := deleteKappnavInstance(ctx, k, c)
	if err != nil {
		return err
	}

	// Delete the deployment.
	clientset, err := getClientset()
	if err != nil {
		return err
	}

	deploymentClient := clientset.AppsV1().Deployments(k.GetNamespace())
	if err != nil {
		return err
	}
	deletePolicy := metav1.DeletePropagationForeground
	err = deploymentClient.Delete("helm-operator", &metav1.DeleteOptions{
		PropagationPolicy: &deletePolicy})
	if err != nil {
		if !apiErrors.IsNotFound(err) {
			return err
		}
	}

	// Delete the service account.
	saClient := clientset.CoreV1().ServiceAccounts(k.GetNamespace())
	err = saClient.Delete("helm-operator", &metav1.DeleteOptions{
		PropagationPolicy: &deletePolicy})

	if err != nil {
		if apiErrors.IsNotFound(err) {
			return nil
		}
	}

	return err
}

// Delete the kAppNav CR instance.
func deleteKappnavInstance(ctx context.Context, k *kabanerov1alpha1.Kabanero, c client.Client) error {
	m, err := mf.NewManifest(crFileName, true, c)
	if err != nil {
		return err
	}

	transforms := []mf.Transformer{
		mf.InjectOwner(k),
		mf.InjectNamespace(k.GetNamespace()),
	}

	err = m.Transform(transforms...)
	if err != nil {
		return err
	}

	err = m.DeleteAll()
	if err != nil {
		return err
	}

	// Make sure the instance is down. This may take a while. Wait for 2 minutes.
	err = kutils.Retry(24, 5*time.Second, func() (bool, error) {
		kappnanvInst := &unstructured.Unstructured{}
		kappnanvInst.SetGroupVersionKind(schema.GroupVersionKind{
			Kind:    "KAppNav",
			Group:   "charts.helm.k8s.io",
			Version: "v1alpha1",
		})

		err = c.Get(ctx, client.ObjectKey{
			Name:      "instance",
			Namespace: k.ObjectMeta.Namespace}, kappnanvInst)

		if err != nil {
			if apiErrors.IsNotFound(err) {
				return true, nil
			}

			return false, err
		}

		// Got an instance. Retry.
		return false, nil
	})

	return err
}

// Returns a Clientset object.
func getClientset() (*kubernetes.Clientset, error) {
	// Create a clientset to drive API operations on resources.
	config, err := clientcmd.BuildConfigFromFlags("", "")
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return clientset, err
}
