package kabaneroplatform

import (
	"context"
	"fmt"
	"strings"

	mf "github.com/jcrossley3/manifestival"
	kabanerov1alpha1 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
	rlog "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var kanlog = rlog.Log.WithName("kabanero-kappnav")

const currentVersion = "0.1.0"

// Reconciles the Kubernetes Application Navigator.
func reconcileKappnav(ctx context.Context, k *kabanerov1alpha1.Kabanero, c client.Client) error {
	logger := kanlog.WithValues("Kabanero instance namespace", k.Namespace, "Kabanero instance Name", k.Name)
	logger.Info("Reconciling KAppNav")

	// Enable path. Apply the needed resource yamls.
	if k.Spec.Kappnav.Enable {
		filename := "config/reconciler/kappnav-operator/kappnav-0.1.0.yaml"
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

		logger.Info(fmt.Sprintf("reconcileKappnav: KAppNav Resources: %v", m))
		err = m.ApplyAll()

		return err
	}

	// Disable path. Do basic cleanup for now to be consistent with kabanero instance cleanup.
	err := processCleanup(k)

	return err
}

// Retrieves the Kubernetes Application Navigator deployment status.
func getKappnavStatus(k *kabanerov1alpha1.Kabanero, c client.Client) (bool, error) {
	// KNavApp is optional. The status was defined as a pointer so that it is not displayed
	// in the kabanero instance data if KAppNav is disabled. That is because structures are
	// never 'empty' for json tagging 'omitempty' to take effect.
	// We need to create the structure here before we use it.
	status := kabanerov1alpha1.KappnavStatus{}
	k.Status.Kappnav = &status
	k.Status.Kappnav.Version = currentVersion
	k.Status.Kappnav.ErrorMessage = ""
	k.Status.Kappnav.Ready = "False"

	clientset, err := getClient()
	if err != nil {
		k.Status.Landing.Ready = "False"
		k.Status.Landing.ErrorMessage = "Failed to obtain client deployments interface."
		return false, err
	}

	deploymentClient := clientset.AppsV1().Deployments(k.GetNamespace())
	deployment, err := deploymentClient.Get("helm-operator", metav1.GetOptions{})

	if err != nil {
		if errors.IsNotFound(err) {
			if !k.Spec.Kappnav.Enable {
				k.Status.Kappnav = nil
				return true, nil
			}
		}
		message := "Unable to retrieve the kappnav helm-operator deployment object."
		kanlog.Error(err, message)
		k.Status.Kappnav.ErrorMessage = message + ": " + err.Error()
		return false, err
	}

	conditions := deployment.Status.Conditions
	// Find the condition type of 'Available' and it's value.
	ready := false
	for _, condition := range conditions {
		if strings.ToLower(string(condition.Type)) == "available" {
			if strings.ToLower(string(condition.Status)) == "true" {
				ready = true
				k.Status.Kappnav.Ready = "True"
			} else {
				k.Status.Kappnav.ErrorMessage = condition.Message
			}

			break
		}
	}

	return ready, err
}

// Performs cleanup processing.
func processCleanup(k *kabanerov1alpha1.Kabanero) error {
	clientset, err := getClient()
	deploymentClient := clientset.AppsV1().Deployments(k.GetNamespace())
	if err != nil {
		return err
	}

	deletePolicy := metav1.DeletePropagationForeground
	err = deploymentClient.Delete("helm-operator", &metav1.DeleteOptions{
		PropagationPolicy: &deletePolicy})
	if err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
	}

	saClient := clientset.CoreV1().ServiceAccounts(k.GetNamespace())
	err = saClient.Delete("helm-operator", &metav1.DeleteOptions{
		PropagationPolicy: &deletePolicy})

	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
	}

	return err
}

// Returns a Clientset object.
func getClient() (*kubernetes.Clientset, error) {
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
