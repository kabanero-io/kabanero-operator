package kabaneroplatform

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-logr/logr"
	kabanerov1alpha1 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha1"
	kutils "github.com/kabanero-io/kabanero-operator/pkg/controller/kabaneroplatform/utils"
	"github.com/kabanero-io/kabanero-operator/pkg/versioning"
	mf "github.com/kabanero-io/manifestival"
	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apiextclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	rlog "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var chelog = rlog.Log.WithName("kabanero-che")

const (
	// Software names as the appear in versions.yaml.
	versionSoftwareNameCheOperator = "che-operator"
	versionSoftwareNameKabaneroChe = "kabanero-che"

	// Yaml file names for deployment.
	yamlNameCheOperatorCRD         = "che-operator-crd.yaml"
	yamlNameCheOperatorDeployments = "che-operator.yaml"
	yamlNameCodewindRoleBinding    = "codewind-role-binding.yaml"
	yamlNameCodewindClusterRole    = "codewind-cluster-role.yaml"
	yamlNameCodewindCheOperatorCR  = "codewind-che-cr.yaml"

	// Deployed Resource names.
	nameCodewindCheOperatorCR = "codewind-che"
)

func initializeChe(k *kabanerov1alpha1.Kabanero) {
	if k.Spec.Che.Enable == nil {
		enable := false
		k.Spec.Che.Enable = &enable
	}
}

func reconcileChe(ctx context.Context, k *kabanerov1alpha1.Kabanero, c client.Client, ctrlr controller.Controller) error {
	// The Che entry was not configured in the spec. Concider Che to be disabled.
	if *k.Spec.Che.Enable == false {
		cleanupChe(ctx, k, c)
		return nil
	}

	logger := chelog.WithValues("Kabanero instance namespace", k.Namespace, "Kabanero instance Name", k.Name)
	logger.Info("Reconciling Che install.")

	// Deploy the Che operator.
	rev, err := resolveSoftwareRevision(k, versionSoftwareNameCheOperator, k.Spec.Che.CheOperator.Version)
	if err != nil {
		logger.Error(err, "Che operator deloyment failed. Unable to resolve softeare revision.")
		return err
	}

	templateCtx := rev.Identifiers
	image, err := imageUriWithOverrides(k.Spec.Che.CheOperator.Repository, k.Spec.Che.CheOperator.Tag, k.Spec.Che.CheOperator.Image, rev)
	if err != nil {
		logger.Error(err, "Che operator deloyment failed. Unable to process image overrides.")
		return err
	}
	templateCtx["image"] = image

	err = processYaml(ctx, k, rev, templateCtx, c, yamlNameCheOperatorCRD, true)
	if err != nil {
		logger.Error(err, fmt.Sprintf("Failed to Apply Che CRD. Revision: %v. TemplateCtx: %v", rev, templateCtx))
		return err
	}

	err = processYaml(ctx, k, rev, templateCtx, c, yamlNameCheOperatorDeployments, true)
	if err != nil {
		logger.Error(err, fmt.Sprintf("Failed to Apply Che operator deployment resource. Revision: %v. TemplateCtx: %v", rev, templateCtx))
		return err
	}

	// Deploy the cluster role with the required permissions for codewind.
	err = processYaml(ctx, k, rev, templateCtx, c, yamlNameCodewindClusterRole, true)
	if err != nil {
		logger.Error(err, fmt.Sprintf("Failed to Apply clusterRole resource. Revision: %v. TemplateCtx: %v", rev, templateCtx))
		return err
	}

	// Deploy the cluster role with the required permissions for codewind.
	err = processYaml(ctx, k, rev, templateCtx, c, yamlNameCodewindRoleBinding, true)
	if err != nil {
		logger.Error(err, fmt.Sprintf("Failed to Apply RoleBinding. Revision: %v. TemplateCtx: %v", rev, templateCtx))
		return err
	}

	// Be sure the Che CRD is active before we deploy an instance.
	crdActive, err := isCheCRDActive()
	if err != nil {
		logger.Error(err, "Failed to verify if Che CRD is active.")
		return err
	}

	// Apply the codewind-che CR instance if it does not already exists.
	if crdActive {
		err = deployCheInstance(ctx, k, c, ctrlr, logger)
		if err != nil {
			logger.Error(err, fmt.Sprintf("Failed to create or validate Che instance. Controller: %v.", ctrlr))
			return err
		}
	}

	return nil
}

// Deploys the Che operator CR if one does not exist. If the Che operator CR exists, it validates that the image and tag values
// are consistent with what was configured.
func deployCheInstance(ctx context.Context, k *kabanerov1alpha1.Kabanero, c client.Client, ctrlr controller.Controller, logger logr.Logger) error {
	deployed, err := isCheInstanceDeployed(ctx, k, c, nameCodewindCheOperatorCR)
	if err != nil {
		return err
	}
	if !deployed {
		// Get the version of the Che Operator. The operator version is kept in sync with the version
		// of eclipse-che deployed by kabanero-che. This version is used to update
		// some fields (devfileRegistryImage, pluginRegistryImage, identityProviderImage) in the
		// Che operator CR deployment yaml to keep in sync with the operator/che versions.
		rev, err := resolveSoftwareRevision(k, versionSoftwareNameCheOperator, k.Spec.Che.CheOperator.Version)
		if err != nil {
			return err
		}
		operatorVersion := rev.Version

		// Get information about Kabanero-Che. Use this version of the revision object for further actions.
		rev, err = resolveSoftwareRevision(k, versionSoftwareNameKabaneroChe, k.Spec.Che.KabaneroChe.Version)
		if err != nil {
			return err
		}
		templateCtx := rev.Identifiers
		image, err := imageUriWithOverrides(k.Spec.Che.CheOperator.Repository, k.Spec.Che.KabaneroChe.Tag, k.Spec.Che.KabaneroChe.Image, rev)
		if err != nil {
			return err
		}
		templateCtx["image"] = image
		templateCtx["workspaceClusterRole"] = getWorkspaceClusterRole(k)
		templateCtx["cheTag"] = operatorVersion

		err = processYaml(ctx, k, rev, templateCtx, c, yamlNameCodewindCheOperatorCR, true)
		if err != nil {
			return err
		}

		// Watch Che instances. TODO: Do this only once if Che use is enabled.
		err = watchCheInstance(ctrlr)
		if err != nil {
			return err
		}
	} else {
		err = validateCheInstance(ctx, k, c)
		if err != nil {
			return err
		}
	}

	return nil
}

// Adds a watch that keeps track of changed to Che instances
func watchCheInstance(ctrlr controller.Controller) error {
	handler := &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &kabanerov1alpha1.Kabanero{},
	}

	predicate := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			return e.MetaOld.GetGeneration() != e.MetaNew.GetGeneration()
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return !e.DeleteStateUnknown
		},
	}

	cheInstance := &unstructured.Unstructured{}
	cheInstance.SetGroupVersionKind(schema.GroupVersionKind{
		Kind:    "CheCluster",
		Group:   "org.eclipse.che",
		Version: "v1",
	})

	err := ctrlr.Watch(&source.Kind{Type: cheInstance}, handler, predicate)
	if err != nil {
		custErr := fmt.Errorf("Unable set a watch for Che instance. Error: %v.  Che instance: %v", err, cheInstance)
		return custErr
	}

	return err
}

// Applies or deletes the specfied yaml file.
func processYaml(ctx context.Context, k *kabanerov1alpha1.Kabanero, rev versioning.SoftwareRevision, templateCtx map[string]interface{}, c client.Client, fileName string, apply bool) error {
	f, err := rev.OpenOrchestration(fileName)
	if err != nil {
		return err
	}

	s, err := renderOrchestration(f, templateCtx)
	if err != nil {
		return err
	}

	m, err := mf.FromReader(strings.NewReader(s), c)
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

	if apply {
		err = m.ApplyAll()
	} else {
		err = m.DeleteAll()
	}

	return err
}

// Returns true if the Che CRD is active. False, otherwise.
func isCheCRDActive() (bool, error) {
	config, err := clientcmd.BuildConfigFromFlags("", "")
	if err != nil {
		return false, err
	}

	extClientset, err := apiextclientset.NewForConfig(config)
	if err != nil {
		return false, err
	}

	err = kutils.Retry(12, 5*time.Second, func() (bool, error) {
		active := false
		crd, err := extClientset.ApiextensionsV1beta1().CustomResourceDefinitions().Get("checlusters.org.eclipse.che", metav1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
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

// Retrieves the Kabanero Che status.
func getCheStatus(ctx context.Context, k *kabanerov1alpha1.Kabanero, c client.Client) (bool, error) {
	// If disabled. Nothing to do. No need to display status if disabled.
	if *k.Spec.Che.Enable == false {
		k.Status.Che = nil
		return true, nil
	}

	// Kabanero Che is optional. The status was defined as a pointer so that it is not displayed
	// in the kabanero instance data if Che is disabled. That is because structures are
	// never 'empty' for json tagging 'omitempty' to take effect.
	// We need to create the structure here before we use it.
	k.Status.Che = &kabanerov1alpha1.CheStatus{}
	k.Status.Che.ErrorMessage = ""
	k.Status.Che.Ready = "False"

	// Retrieve the Che operator version being used.
	rev, err := resolveSoftwareRevision(k, versionSoftwareNameCheOperator, k.Spec.Che.CheOperator.Version)
	if err != nil {
		return false, err
	}

	k.Status.Che.CheOperator.Version = rev.Version

	// Retrieve the kabanero Che version being used.
	rev, err = resolveSoftwareRevision(k, versionSoftwareNameKabaneroChe, k.Spec.Che.KabaneroChe.Version)
	if err != nil {
		return false, err
	}

	image, err := imageUriWithOverrides(k.Spec.Che.CheOperator.Repository, k.Spec.Che.KabaneroChe.Tag, k.Spec.Che.KabaneroChe.Image, rev)
	if err != nil {
		return false, err
	}

	imageParts := strings.Split(image, ":")
	if len(imageParts) != 2 {
		return false, fmt.Errorf("Image %v is not valid", image)
	}

	k.Status.Che.KabaneroChe.Version = rev.Version
	k.Status.Che.KabaneroCheInstance.CheImage = imageParts[0]
	k.Status.Che.KabaneroCheInstance.CheImageTag = imageParts[1]
	k.Status.Che.KabaneroCheInstance.CheWorkspaceClusterRole = getWorkspaceClusterRole(k)

	// Get Kabanero Che instance to discern the state of the Che installation.
	cheInst, err := getCheInstance(ctx, k, c, nameCodewindCheOperatorCR)

	if err != nil {
		custErr := fmt.Errorf("Unable to retrieve the Che instance object. Error: %v", err)
		k.Status.Che.ErrorMessage = custErr.Error()
		return false, custErr
	}

	// Get the cheClusterRunning status from the resource.
	cheClusterRunning, ok, err := unstructured.NestedString(cheInst.Object, "status", "cheClusterRunning")
	if err != nil || !ok {
		custErr := fmt.Errorf("Unable to retrieve status.cheClusterRunning from the Che instance, Error: %v", err)
		k.Status.Che.ErrorMessage = custErr.Error()
		return false, custErr
	}

	ready := false
	if cheClusterRunning == "Available" {
		ready = true
		k.Status.Che.Ready = "True"
	} else {
		k.Status.Che.ErrorMessage = cheClusterRunning
	}

	return ready, nil
}

// Performs cleanup processing.
func cleanupChe(ctx context.Context, k *kabanerov1alpha1.Kabanero, c client.Client) error {
	// Delete the CR instance. All other Che operator namespaced resources will be deleted
	// when the kabanero CR instance is deleted.
	err := deleteCheInstance(ctx, k, c)
	if err != nil {
		return err
	}

	// If the CR instance was deleted, delete the Che operator.
	err = deleteCheOperatorResources(ctx, k, c)
	if err != nil {
		return err
	}

	return nil
}

// Delete the Kabanero Che operator deployment yamls.
func deleteCheOperatorResources(ctx context.Context, k *kabanerov1alpha1.Kabanero, c client.Client) error {
	rev, err := resolveSoftwareRevision(k, versionSoftwareNameCheOperator, k.Spec.Che.CheOperator.Version)
	if err != nil {
		return err
	}

	templateCtx := rev.Identifiers
	image, err := imageUriWithOverrides(k.Spec.Che.CheOperator.Repository, k.Spec.Che.CheOperator.Tag, k.Spec.Che.CheOperator.Image, rev)
	if err != nil {
		return err
	}

	templateCtx["image"] = image

	err = processYaml(ctx, k, rev, templateCtx, c, yamlNameCodewindRoleBinding, false)
	if err != nil {
		return err
	}

	err = processYaml(ctx, k, rev, templateCtx, c, yamlNameCheOperatorDeployments, false)
	if err != nil {
		return err
	}

	return nil
}

// Delete the Kabanero Che CR instance.
func deleteCheInstance(ctx context.Context, k *kabanerov1alpha1.Kabanero, c client.Client) error {
	rev, err := resolveSoftwareRevision(k, versionSoftwareNameCheOperator, k.Spec.Che.CheOperator.Version)
	if err != nil {
		return err
	}

	templateCtx := rev.Identifiers
	image, err := imageUriWithOverrides(k.Spec.Che.CheOperator.Repository, k.Spec.Che.CheOperator.Tag, k.Spec.Che.CheOperator.Image, rev)
	if err != nil {
		return err
	}

	templateCtx["image"] = image

	err = processYaml(ctx, k, rev, templateCtx, c, yamlNameCodewindCheOperatorCR, false)
	if err != nil {
		return err
	}

	// Make sure the instance is down. This may take a while. Wait for 2 minutes.
	err = kutils.Retry(24, 5*time.Second, func() (bool, error) {
		deployed, err := isCheInstanceDeployed(ctx, k, c, nameCodewindCheOperatorCR)

		if err != nil {
			return false, err
		}

		if !deployed {
			return true, nil
		}

		// Got an instance. Retry.
		return false, nil
	})

	if err != nil {
		return err
	}

	return nil
}

// Get the named Kabanero Che instance object as unstructured data
func getCheInstance(ctx context.Context, k *kabanerov1alpha1.Kabanero, c client.Client, instanceName string) (*unstructured.Unstructured, error) {
	cheInst := &unstructured.Unstructured{}
	cheInst.SetGroupVersionKind(schema.GroupVersionKind{
		Kind:    "CheCluster",
		Group:   "org.eclipse.che",
		Version: "v1",
	})

	err := c.Get(ctx, client.ObjectKey{
		Name:      instanceName,
		Namespace: k.ObjectMeta.Namespace}, cheInst)

	return cheInst, err
}

func isCheInstanceDeployed(ctx context.Context, k *kabanerov1alpha1.Kabanero, c client.Client, instanceName string) (bool, error) {
	_, err := getCheInstance(ctx, k, c, instanceName)

	if err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

// Validates image information for the deployed instance. Entries not matching the config are updated.
func validateCheInstance(ctx context.Context, k *kabanerov1alpha1.Kabanero, c client.Client) error {
	// Get Kabanero Che instance.
	cheInst, err := getCheInstance(ctx, k, c, nameCodewindCheOperatorCR)

	if err != nil {
		custErr := fmt.Errorf("Unable to retrieve the Che instance object. Error: %v", err)
		return custErr
	}

	// Load the Che instance object spec.server entry.
	server, ok, err := unstructured.NestedFieldCopy(cheInst.Object, "spec", "server")
	if err != nil || !ok {
		custErr := fmt.Errorf("Unable to retrieve spec.server from the Che instance, Error: %v", err)
		return custErr
	}
	if server == nil {
		custErr := fmt.Errorf("Retrieve a nil spec.server from the Che instance, Error: %v", err)
		return custErr
	}

	serverOptions, ok := server.(map[string]interface{})

	if !ok {
		custErr := fmt.Errorf("Error casting server options into the appropriate type")
		return custErr
	}

	// Get the data needed for validation: repository image and image Tag.
	rev, err := resolveSoftwareRevision(k, versionSoftwareNameKabaneroChe, k.Spec.Che.KabaneroChe.Version)
	if err != nil {
		return err
	}

	image, err := imageUriWithOverrides(k.Spec.Che.KabaneroChe.Repository, k.Spec.Che.KabaneroChe.Tag, k.Spec.Che.KabaneroChe.Image, rev)
	if err != nil {
		return err
	}

	imageParts := strings.Split(image, ":")
	if len(imageParts) != 2 {
		return fmt.Errorf("Image %v is not valid", image)
	}

	// Validate that the repository image and image tag are what we expect it to be.
	// If the user updated this information, replace it is configured.
	imageChecked := false
	tagChecked := false
	for key, val := range serverOptions {
		if key == "cheImage" {
			if val != imageParts[0] {
				unstructured.SetNestedField(cheInst.Object, imageParts[0], "spec", "server", "cheImage")
				imageChecked = true
			}
		} else if key == "cheImageTag" {
			if val != imageParts[1] {
				unstructured.SetNestedField(cheInst.Object, imageParts[1], "spec", "server", "cheImageTag")
				tagChecked = true
			}
		} else if key == "cheWorkspaceClusterRole" {
			wscr := getWorkspaceClusterRole(k)
			if val != wscr {
				unstructured.SetNestedField(cheInst.Object, wscr, "spec", "server", "cheWorkspaceClusterRole")
				tagChecked = true
			}
		}

		if imageChecked && tagChecked {
			break
		}
	}

	// Update the instance.
	err = c.Update(ctx, cheInst)
	if err != nil {
		return err
	}

	return nil
}

// Returns the workspaceClusterRole value to be used when deploying the Che CR instance.
// Users may choose to override this value by specifying it in the kabanero CR instance yaml.
func getWorkspaceClusterRole(k *kabanerov1alpha1.Kabanero) string {
	wscr := k.Spec.Che.CheOperatorInstance.CheWorkspaceClusterRole
	if len(wscr) == 0 {
		wscr = "eclipse-codewind"
	}

	return wscr
}
