package kabaneroplatform

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-logr/logr"
	kabanerov1alpha2 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha2"
	kutils "github.com/kabanero-io/kabanero-operator/pkg/controller/kabaneroplatform/utils"
	"github.com/kabanero-io/kabanero-operator/pkg/controller/utils/timer"
	"github.com/kabanero-io/kabanero-operator/pkg/versioning"
	mfc "github.com/manifestival/controller-runtime-client"
	mf "github.com/manifestival/manifestival"
	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apiextclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	rlog "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var crwlog = rlog.Log.WithName("kabanero-codeready-workspaces")

const (
	crwOrchestrationFilePath         = "orchestrations/codeready-workspaces/0.1"
	crwYamlNameCodewindClusterRole   = "codewind-clusterrole.yaml"
	crwYamlNameCodewindTektonRole    = "codewind-tekton-role.yaml"
	crwYamlNameCodewindTektonBinding = "codewind-tekton-rolebinding.yaml"
	crwOperatorCR                    = "codeready-workspaces-cr.yaml"
	crwOperatorCRNameSuffix          = "codeready-workspaces"
	crwVersionSoftwareName           = "codeready-workspaces"
	crwOperatorSubscriptionName      = "codeready-workspaces"

	crwVersionOrchDevfileRegRepository = "devfile-reg-repository"
	crwVersionOrchDevfileRegTag        = "devfile-reg-tag"
)

func initializeCRW(k *kabanerov1alpha2.Kabanero) {
	if k.Spec.CodereadyWorkspaces.Enable == nil {
		enable := false
		k.Spec.CodereadyWorkspaces.Enable = &enable
	}
}

func reconcileCRW(ctx context.Context, k *kabanerov1alpha2.Kabanero, c client.Client, reqLogger logr.Logger) error {
	logger := crwlog.WithValues("Kabanero instance namespace", k.Namespace, "Kabanero instance Name", k.Name)
	logger.Info("Reconciling codeready-workspaces install.")

	rev, err := resolveSoftwareRevision(k, "codeready-workspaces", k.Spec.CodereadyWorkspaces.Operator.CustomResourceInstance.DevFileRegistryImage.Version)
	if err != nil {
		return err
	}

	// The Che entry was not configured in the spec. Consider Che to be disabled.
	if *k.Spec.CodereadyWorkspaces.Enable == false {
		cleanupCRW(ctx, k, rev, c)
		return nil
	}

	templateCtx := unstructured.Unstructured{}.Object

	// Deploy the Codewind cluster role with the required permissions for codewind.
	err = processCRWYaml(ctx, k, rev, templateCtx, c, crwYamlNameCodewindClusterRole, true, k.GetNamespace())
	if err != nil {
		logger.Error(err, fmt.Sprintf("Failed to Apply clusterRole resource. Revision: %v. TemplateCtx: %v", rev, templateCtx))
		return err
	}

	// Deploy the Codewind Tekton role
	err = processCRWYaml(ctx, k, rev, templateCtx, c, crwYamlNameCodewindTektonRole, true, "tekton-pipelines")
	if err != nil {
		logger.Error(err, fmt.Sprintf("Failed to Apply role resource. Revision: %v. TemplateCtx: %v", rev, templateCtx))
		return err
	}

	// Deploy the Codewind Tekton rolebinding
	err = processCRWYaml(ctx, k, rev, templateCtx, c, crwYamlNameCodewindTektonBinding, true, "tekton-pipelines")
	if err != nil {
		logger.Error(err, fmt.Sprintf("Failed to Apply rolebinding resource. Revision: %v. TemplateCtx: %v", rev, templateCtx))
		return err
	}

	// Be sure the codeready-workspaces CRD is active before we deploy an instance.
	crdActive, err := isCRWCRDActive()
	if err != nil {
		logger.Error(err, "Failed to verify if the codeready-workspaces CRD is active.")
		return err
	}

	// Apply the codeready-workspaces CR instance if it does not already exists.
	if crdActive {
		err = deployCRWInstance(ctx, k, c, rev, logger)
		if err != nil {
			logger.Error(err, fmt.Sprintf("Failed to create or validate codeready-workspaces instance. Controller: %v.", ctrlr))
			return err
		}
	}

	return nil
}

// Deploys the codeready-workspaces operator CR if one does not exist. If the codeready-workspaces operator CR exists,
// it validates that the image and tag values are consistent with what was configured.
func deployCRWInstance(ctx context.Context, k *kabanerov1alpha2.Kabanero, c client.Client, rev versioning.SoftwareRevision, logger logr.Logger) error {
	deployed, err := isCRWInstanceDeployed(ctx, k, c)
	if err != nil {
		return err
	}

	// Deploy the instance once and watch it. Further updates to entries need to be done individually because
	// instance's entries are automatically populated  with default values. We do not want to
	// override them by applying the cr instance yaml file more than once.
	if !deployed {
		templateCtx, err := getCRWInstanceOrchestrationTemplate(k, rev)
		if err != nil {
			return err
		}

		err = processCRWYaml(ctx, k, rev, templateCtx, c, crwOperatorCR, true, k.GetNamespace())
		if err != nil {
			return err
		}
	} else {
		validateCRWInstance(ctx, k, c, rev)
		if err != nil {
			return err
		}
	}

	return nil
}

// Adds a watch that keeps track of changed to *-codeready-workspaces (CheCluster) instances
func watchCRWInstance(ctrlr controller.Controller) error {
	handler := &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &kabanerov1alpha2.Kabanero{},
	}

	predicate := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			return e.MetaOld.GetGeneration() != e.MetaNew.GetGeneration()
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return !e.DeleteStateUnknown
		},
	}

	crwInstance := &unstructured.Unstructured{}
	crwInstance.SetGroupVersionKind(schema.GroupVersionKind{
		Kind:    "CheCluster",
		Group:   "org.eclipse.che",
		Version: "v1",
	})

	err := ctrlr.Watch(&source.Kind{Type: crwInstance}, handler, predicate)
	if err != nil {
		custErr := fmt.Errorf("Unable set a watch for the codeready-workspaces instance. Error: %v. Codeready-workspaces instance: %v", err, crwInstance)
		return custErr
	}

	return err
}

// Validates that the kabanero CR configured values for codeready-workspaces do not change.
func validateCRWInstance(ctx context.Context, k *kabanerov1alpha2.Kabanero, c client.Client, rev versioning.SoftwareRevision) error {
	// Get Kabanero codeready-workspaces instance.
	crwInst, err := getCRWInstance(ctx, k, c)

	if err != nil {
		custErr := fmt.Errorf("Unable to retrieve the codeready-workspaces instance object. Error: %v", err)
		return custErr
	}

	// Load the code-ready instance spec.server entry.
	server, found, err := unstructured.NestedFieldCopy(crwInst.Object, "spec", "server")
	if err != nil {
		custErr := fmt.Errorf("Unable to retrieve spec.server from the codeready-workspaces instance, Error: %v", err)
		return custErr
	}
	if !found {
		custErr := fmt.Errorf("The value of codeready-workspaces instance entry spec.server was not found, Error: %v", err)
		return custErr
	}

	serverOptions, ok := server.(map[string]interface{})

	if !ok {
		custErr := fmt.Errorf("Error casting server options into the appropriate type")
		return custErr
	}

	// Validate that the kavanero CR configured entries are what we expect it to be.
	// If the user updated this information on the console, replace it with the kabanero CR
	// configured values.
	kssc := getCRWCRInstanceBoolean(k.Spec.CodereadyWorkspaces.Operator.CustomResourceInstance.SelfSignedCert)
	ssc, exists := serverOptions["selfSignedCert"]
	if !exists {
		ssc = kssc
	}
	if ssc != kssc {
		err := unstructured.SetNestedField(crwInst.Object, kssc, "spec", "server", "selfSignedCert")
		if err != nil {
			return err
		}
	}

	ktlss := getCRWCRInstanceBoolean(k.Spec.CodereadyWorkspaces.Operator.CustomResourceInstance.TLSSupport)
	tlss, exists := serverOptions["tlsSupport"]
	if !exists {
		tlss = ktlss
	}
	if tlss != ktlss {
		err := unstructured.SetNestedField(crwInst.Object, ktlss, "spec", "server", "tlsSupport")
		if err != nil {
			return err
		}
	}

	cwscr, exists := serverOptions["cheWorkspaceClusterRole"]
	if !exists {
		cwscr = k.Spec.CodereadyWorkspaces.Operator.CustomResourceInstance.CheWorkspaceClusterRole
	}
	if cwscr != k.Spec.CodereadyWorkspaces.Operator.CustomResourceInstance.CheWorkspaceClusterRole {
		err := unstructured.SetNestedField(crwInst.Object, k.Spec.CodereadyWorkspaces.Operator.CustomResourceInstance.CheWorkspaceClusterRole, "spec", "server", "cheWorkspaceClusterRole")
		if err != nil {
			return err
		}
	}

	kdfri, err := getCRWCRDevfileRegistryImage(k, rev)
	if err != nil {
		return err
	}
	dfri, exists := serverOptions["devfileRegistryImage"]

	if !exists {
		dfri = kdfri
	}

	if dfri != kdfri {
		err := unstructured.SetNestedField(crwInst.Object, kdfri, "spec", "server", "devfileRegistryImage")
		if err != nil {
			return err
		}
	}

	// Load the codeready-workspaces instance spec.auth entry.
	auth, ok, err := unstructured.NestedFieldCopy(crwInst.Object, "spec", "auth")
	if err != nil || !ok {
		custErr := fmt.Errorf("Unable to retrieve spec.auth from the codeready-workspaces instance, Error: %v", err)
		return custErr
	}
	if server == nil {
		custErr := fmt.Errorf("Retrieve a nil spec.auth from the codeready-workspaces instance, Error: %v", err)
		return custErr
	}

	authOptions, ok := auth.(map[string]interface{})

	if !ok {
		custErr := fmt.Errorf("Error casting auth options into the appropriate type")
		return custErr
	}

	// Validate that openShiftoAuth is what we expect it to be.
	// If the user updated this information, replace it if configured.
	kosoa := getCRWCRInstanceBoolean(k.Spec.CodereadyWorkspaces.Operator.CustomResourceInstance.OpenShiftOAuth)
	osoa, exists := authOptions["openShiftoAuth"]
	if !exists {
		osoa = kosoa
	}
	if osoa != kosoa {
		err := unstructured.SetNestedField(crwInst.Object, kosoa, "spec", "auth", "openShiftoAuth")
		if err != nil {
			return err
		}
	}

	// Update the instance.
	err = c.Update(ctx, crwInst)
	if err != nil {
		return err
	}

	return nil
}

// Applies or deletes the specified yaml file.
func processCRWYaml(ctx context.Context, k *kabanerov1alpha2.Kabanero, rev versioning.SoftwareRevision, templateCtx map[string]interface{}, c client.Client, fileName string, apply bool, namespace string) error {
	f, err := rev.OpenOrchestration(fileName)
	if err != nil {
		return err
	}

	s, err := renderOrchestration(f, templateCtx)
	if err != nil {
		return err
	}

	mOrig, err := mf.ManifestFrom(mf.Reader(strings.NewReader(s)), mf.UseClient(mfc.NewClient(c)), mf.UseLogger(rlog.Log.WithName("manifestival")))
	if err != nil {
		return err
	}

	transforms := []mf.Transformer{
		mf.InjectOwner(k),
		mf.InjectNamespace(namespace),
	}

	m, err := mOrig.Transform(transforms...)
	if err != nil {
		return err
	}

	if apply {
		err = m.Apply()
	} else {
		err = m.Delete()
	}

	return err
}

// Returns true if the codeready-workspaces CRD is active. False, otherwise.
func isCRWCRDActive() (bool, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return false, err
	}

	extClientset, err := apiextclientset.NewForConfig(config)
	if err != nil {
		return false, err
	}

	err = timer.Retry(12, 5*time.Second, func() (bool, error) {
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

// Retrieves codeready-workspaces status.
func getCRWStatus(ctx context.Context, k *kabanerov1alpha2.Kabanero, c client.Client) (bool, error) {
	// If disabled. Nothing to do. No need to display status if disabled.
	if *k.Spec.CodereadyWorkspaces.Enable == false {
		k.Status.CodereadyWorkspaces = nil
		return true, nil
	}

	// Codeready-workspaces operator deployment is optional. The status was defined as a pointer so that it is not displayed
	// in the kabanero instance data if codeready-workspaces is disabled. That is because structures are
	// never 'empty' for json tagging 'omitempty' to take effect.
	// We need to create the structure here before we use it.
	k.Status.CodereadyWorkspaces = &kabanerov1alpha2.CRWStatus{}

	k.Status.CodereadyWorkspaces.Ready = "False"
	k.Status.CodereadyWorkspaces.Message = ""

	// Retrieve the version of the codeready-workspaces operator.
	crwOperatorVersion, err := getCRWOperatorVersion(k, c)
	if err != nil {
		k.Status.CodereadyWorkspaces.Message = "Unable to retrieve the version of installed codeready-workspaces operator. Error: " + err.Error()
		return false, err
	}

	k.Status.CodereadyWorkspaces.Operator.Version = crwOperatorVersion

	// Get the codeready-workspaces instance.
	crwInst, err := getCRWInstance(ctx, k, c)
	if err != nil {
		custErr := fmt.Errorf("Unable to retrieve the codeready-workspaces instance object. Error: %v", err)
		k.Status.CodereadyWorkspaces.Message = custErr.Error()
		return false, custErr
	}

	// Get the cheClusterRunning status from the resource.
	cheClusterRunning, found, err := unstructured.NestedString(crwInst.Object, "status", "cheClusterRunning")
	if err != nil {
		custErr := fmt.Errorf("Unable to retrieve status.cheClusterRunning from the codeready-workspaces instance, Error: %v", err)
		k.Status.CodereadyWorkspaces.Message = custErr.Error()
		return false, custErr
	}

	if !found {
		custErr := fmt.Errorf("The value of codeready-workspaces instance entry status.cheClusterRunning was not found")
		k.Status.CodereadyWorkspaces.Message = custErr.Error()
		return false, custErr
	}

	rev, err := resolveSoftwareRevision(k, "codeready-workspaces", k.Spec.CodereadyWorkspaces.Operator.CustomResourceInstance.DevFileRegistryImage.Version)
	if err != nil {
		return false, err
	}

	dfrImage, err := getCRWCRDevfileRegistryImage(k, rev)
	if err != nil {
		return false, err
	}
	k.Status.CodereadyWorkspaces.Operator.Instance.DevfileRegistryImage = dfrImage

	k.Status.CodereadyWorkspaces.Operator.Instance.CheWorkspaceClusterRole = getCRWClusterRole(k)
	k.Status.CodereadyWorkspaces.Operator.Instance.OpenShiftOAuth = getCRWCRInstanceBoolean(k.Spec.CodereadyWorkspaces.Operator.CustomResourceInstance.OpenShiftOAuth)
	k.Status.CodereadyWorkspaces.Operator.Instance.SelfSignedCert = getCRWCRInstanceBoolean(k.Spec.CodereadyWorkspaces.Operator.CustomResourceInstance.SelfSignedCert)
	k.Status.CodereadyWorkspaces.Operator.Instance.TLSSupport = getCRWCRInstanceBoolean(k.Spec.CodereadyWorkspaces.Operator.CustomResourceInstance.TLSSupport)

	ready := false
	if cheClusterRunning == "Available" {
		ready = true
		k.Status.CodereadyWorkspaces.Ready = "True"
	} else {
		k.Status.CodereadyWorkspaces.Message = cheClusterRunning
	}

	return ready, nil
}

// Performs codeready-workspaces cleanup processing.
func cleanupCRW(ctx context.Context, k *kabanerov1alpha2.Kabanero, rev versioning.SoftwareRevision, c client.Client) error {
	// Delete the CR instance.
	err := deleteCRWInstance(ctx, k, rev, c)
	if err != nil {
		return err
	}

	// If the CR instance was deleted, delete the codeready-workspaces operator.
	err = deleteCRWOperatorResources(ctx, k, c)
	if err != nil {
		return err
	}

	return nil
}

// Deletes the resources associated with the codeready-workspaces deployment.
func deleteCRWOperatorResources(ctx context.Context, k *kabanerov1alpha2.Kabanero, c client.Client) error {
	rev, err := resolveSoftwareRevision(k, "codeready-workspaces", k.Spec.CodereadyWorkspaces.Operator.CustomResourceInstance.DevFileRegistryImage.Version)
	if err != nil {
		return err
	}
	kiList := &kabanerov1alpha2.KabaneroList{}
	err = c.List(context.TODO(), kiList)
	if err != nil {
		return err
	}

	if len(kiList.Items) == 1 {
		err = processCRWYaml(ctx, k, rev, unstructured.Unstructured{}.Object, c, crwYamlNameCodewindClusterRole, false, k.GetNamespace())
		if err != nil {
			return err
		}

		// Delete the Tekton role and rolebinding too
		err = processCRWYaml(ctx, k, rev, unstructured.Unstructured{}.Object, c, crwYamlNameCodewindTektonRole, false, "tekton-pipelines")
		if err != nil {
			return err
		}
		err = processCRWYaml(ctx, k, rev, unstructured.Unstructured{}.Object, c, crwYamlNameCodewindTektonBinding, false, "tekton-pipelines")
		if err != nil {
			return err
		}
	}
	return nil
}

// Deletes the codeready-workspaces CR instance deployed by the input kabanero CR instance.
func deleteCRWInstance(ctx context.Context, k *kabanerov1alpha2.Kabanero, rev versioning.SoftwareRevision, c client.Client) error {
	templateCtx, err := getCRWInstanceOrchestrationTemplate(k, rev)
	if err != nil {
		return err
	}

	err = processCRWYaml(ctx, k, rev, templateCtx, c, crwOperatorCR, false, k.GetNamespace())
	if err != nil {
		return err
	}

	// Make sure the instance is down. This may take a while. Wait for 2 minutes.
	err = timer.Retry(24, 5*time.Second, func() (bool, error) {
		deployed, err := isCRWInstanceDeployed(ctx, k, c)

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

// Retrieves the codeready-workspaces instance deployed by the input kabanero CR instance.
func getCRWInstance(ctx context.Context, k *kabanerov1alpha2.Kabanero, c client.Client) (*unstructured.Unstructured, error) {
	crwInst := &unstructured.Unstructured{}
	crwInst.SetGroupVersionKind(schema.GroupVersionKind{
		Kind:    "CheCluster",
		Group:   "org.eclipse.che",
		Version: "v1",
	})

	err := c.Get(ctx, client.ObjectKey{
		Name:      crwOperatorCRNameSuffix,
		Namespace: k.ObjectMeta.Namespace}, crwInst)

	return crwInst, err
}

// Returns true if the coderead-workspaces instance is found. False, otherwise.
func isCRWInstanceDeployed(ctx context.Context, k *kabanerov1alpha2.Kabanero, c client.Client) (bool, error) {
	_, err := getCRWInstance(ctx, k, c)

	if err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

// Returns the installed codeready-workspaces operator version.
func getCRWOperatorVersion(k *kabanerov1alpha2.Kabanero, c client.Client) (string, error) {
	cok := client.ObjectKey{
		Namespace: k.Namespace,
		Name:      crwOperatorSubscriptionName}

	installedCSVName, err := kutils.GetInstalledCSVName(c, cok)
	if err != nil {
		return "", err
	}

	cok = client.ObjectKey{
		Namespace: k.Namespace,
		Name:      installedCSVName}

	csvVersion, err := kutils.GetCSVSpecVersion(c, cok)
	if err != nil {
		return "", err
	}

	return csvVersion, nil
}

// Returns the spec.server.devfileRegistryImage value to be used when deploying an instance of the codeready-workspaces CR.
func getCRWCRDevfileRegistryImage(k *kabanerov1alpha2.Kabanero, rev versioning.SoftwareRevision) (string, error) {
	dfrImage, err := customImageUriWithOverrides(k.Spec.CodereadyWorkspaces.Operator.CustomResourceInstance.DevFileRegistryImage.Repository,
		k.Spec.CodereadyWorkspaces.Operator.CustomResourceInstance.DevFileRegistryImage.Tag,
		k.Spec.CodereadyWorkspaces.Operator.CustomResourceInstance.DevFileRegistryImage.Image,
		rev,
		crwVersionOrchDevfileRegRepository,
		crwVersionOrchDevfileRegTag)
	if err != nil {
		return "", err
	}

	return dfrImage, nil
}

// Returns the spec.server.cheWorkspaceClusterRole value to be used when deploying an instance of the codeready-workspaces CR.
func getCRWClusterRole(k *kabanerov1alpha2.Kabanero) string {
	crwcr := k.Spec.CodereadyWorkspaces.Operator.CustomResourceInstance.CheWorkspaceClusterRole
	if len(crwcr) == 0 {
		crwcr = "kabanero-codewind"
	}
	return crwcr
}

// Processes a boolean pointer. If the pointer is null, this function returns false. Otherwise, it returns
// the value pointed by the boolean pointer.
func getCRWCRInstanceBoolean(value *bool) bool {
	newVal := value
	if newVal == nil {
		defaultVal := false
		newVal = &defaultVal
	}
	return *newVal
}

// Returns a populated orchestration template for a codeready-workspaces custom resource to be deployed.
func getCRWInstanceOrchestrationTemplate(k *kabanerov1alpha2.Kabanero, rev versioning.SoftwareRevision) (map[string]interface{}, error) {
	templateCtx := rev.Identifiers

	dfrImage, err := getCRWCRDevfileRegistryImage(k, rev)
	if err != nil {
		return templateCtx, err
	}

	templateCtx["kabaneroInstanceName"] = k.ObjectMeta.GetName()
	templateCtx["devfileRegistryImage"] = dfrImage
	templateCtx["cheWorkspaceClusterRole"] = getCRWClusterRole(k)
	templateCtx["selfSignedCert"] = getCRWCRInstanceBoolean(k.Spec.CodereadyWorkspaces.Operator.CustomResourceInstance.SelfSignedCert)
	templateCtx["tlsSupport"] = getCRWCRInstanceBoolean(k.Spec.CodereadyWorkspaces.Operator.CustomResourceInstance.TLSSupport)
	templateCtx["openShiftOAuth"] = getCRWCRInstanceBoolean(k.Spec.CodereadyWorkspaces.Operator.CustomResourceInstance.OpenShiftOAuth)

	return templateCtx, nil
}
