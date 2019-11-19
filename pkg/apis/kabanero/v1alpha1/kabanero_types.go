package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// +kubebuilder:subresource:status

// KabaneroSpec defines the desired state of Kabanero
// +k8s:openapi-gen=true
type KabaneroSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html

	Version string `json:"version,omitempty"`

	TargetNamespaces []string `json:"targetNamespaces,omitempty"`

	Github GithubConfig `json:"github,omitempty"`

	Collections InstanceCollectionConfig `json:"collections,omitempty"`

	Tekton TektonCustomizationSpec `json:"tekton,omitempty"`

	CliServices KabaneroCliServicesCustomizationSpec `json:"cliServices,omitempty"`

	Landing KabaneroLandingCustomizationSpec `json:"landing,omitempty"`

	Che CheCustomizationSpec `json:"che,omitempty"`

	Webhook WebhookCustomizationSpec `json:"webhook,omitempty"`

	AdmissionControllerWebhook AdmissionControllerWebhookCustomizationSpec `json:"admissionControllerWebhook,omitempty"`
}

// InstanceCollectionConfig defines the customization entries for a set of collections.
type InstanceCollectionConfig struct {
	Repositories []RepositoryConfig `json:"repositories,omitempty"`
}

// GithubConfig represents the Github information (public or GHE) where
// the organization and teams managing the collections live.  Members
// of the specified team in the specified organization will have admin
// authority in the Kabanero CLI.
type GithubConfig struct {
	Organization string   `json:"organization,omitempty"`
	Teams        []string `json:"teams,omitempty"`
	ApiUrl       string   `json:"apiUrl,omitempty"`
}

// RepositoryConfig defines customization entries for a collection.
type RepositoryConfig struct {
	Name                       string `json:"name,omitempty"`
	Url                        string `json:"url,omitempty"`
	ActivateDefaultCollections bool   `json:"activateDefaultCollections,omitempty"`
	SkipCertVerification       bool   `json:"skipCertVerification,omitempty"`
}

// TektonCustomizationSpec defines customization entries for Tekton
type TektonCustomizationSpec struct {
	Disabled bool   `json:"disabled,omitempty"`
	Version  string `json:"version,omitempty"`
}

// KabaneroCliServicesCustomizationSpec defines customization entries for the Kabanero CLI.
type KabaneroCliServicesCustomizationSpec struct {
	//Future: Enable     bool   `json:"enable,omitempty"`
	Version                  string `json:"version,omitempty"`
	Image                    string `json:"image,omitempty"`
	Repository               string `json:"repository,omitempty"`
	Tag                      string `json:"tag,omitempty"`
	SessionExpirationSeconds string `json:"sessionExpirationSeconds,omitempty"`
}

// KabaneroLandingCustomizationSpec defines customization entries for Kabanero landing page.
type KabaneroLandingCustomizationSpec struct {
	Enable  *bool  `json:"enable,omitempty"`
	Version string `json:"version,omitempty"`
}

// CheCustomizationSpec defines customization entries for Che.
type CheCustomizationSpec struct {
	Enable              *bool                   `json:"enable,omitempty"`
	CheOperatorInstance CheOperatorInstanceSpec `json:"cheOperatorInstance,omitempty"`
	KabaneroChe         KabaneroCheSpec         `json:"kabaneroChe,omitempty"`
}

// CheOperatorInstanceSpec defines customization entries for the Che operator instance.
type CheOperatorInstanceSpec struct {
	CheWorkspaceClusterRole string `json:"cheWorkspaceClusterRole,omitempty"`
}

// KabaneroCheSpec defines customization entries for Kabanero Che.
type KabaneroCheSpec struct {
	Version    string `json:"version,omitempty"`
	Image      string `json:"image,omitempty"`
	Repository string `json:"repository,omitempty"`
	Tag        string `json:"tag,omitempty"`
}

type WebhookCustomizationSpec struct {
	Enable     bool   `json:"enable,omitempty"`
	Version    string `json:"version,omitempty"`
	Image      string `json:"image,omitempty"`
	Repository string `json:"repository,omitempty"`
	Tag        string `json:"tag,omitempty"`
}

type AdmissionControllerWebhookCustomizationSpec struct {
	Version    string `json:"version,omitempty"`
	Image      string `json:"image,omitempty"`
	Repository string `json:"repository,omitempty"`
	Tag        string `json:"tag,omitempty"`
}

// KabaneroStatus defines the observed state of the Kabanero instance.
// +k8s:openapi-gen=true
type KabaneroStatus struct {
	// Kabanero operator instance readiness status. The status is directly correlated to the availability of resources dependencies.
	KabaneroInstance KabaneroInstanceStatus `json:"kabaneroInstance,omitempty"`

	// Knative eventing instance readiness status.
	KnativeEventing KnativeEventingStatus `json:"knativeEventing,omitempty"`

	// OpenShift serverless operator status.
	Serverless ServerlessStatus `json:"serverless,omitempty"`

	// Tekton instance readiness status.
	Tekton TektonStatus `json:"tekton,omitempty"`

	// CLI readiness status.
	Cli CliStatus `json:"cli,omitempty"`

	// Kabanero Landing page readiness status.
	Landing *KabaneroLandingPageStatus `json:"landing,omitempty"`

	// Appsody instance readiness status.
	Appsody AppsodyStatus `json:"appsody,omitempty"`

	// Kabanero Application Navigator instance readiness status.
	Kappnav *KappnavStatus `json:"kappnav,omitempty"`

	// Che instance readiness status.
	Che *CheStatus `json:"che,omitempty"`

	// Webhook instance status
	Webhook *WebhookStatus `json:"webhook,omitempty"`

	// Admission webhook instance status
	AdmissionControllerWebhook AdmissionControllerWebhookStatus `json:"admissionControllerWebhook,omitempty"`
}

// KabaneroInstanceStatus defines the observed status details of Kabanero operator instance
type KabaneroInstanceStatus struct {
	Ready        string `json:"ready,omitempty"`
	ErrorMessage string `json:"errorMessage,omitempty"`
	Version      string `json:"version,omitempty"`
}

// TektonStatus defines the observed status details of Tekton.
type TektonStatus struct {
	Ready        string `json:"ready,omitempty"`
	ErrorMessage string `json:"errorMessage,omitempty"`
	Version      string `json:"version,omitempty"`
}

// KnativeEventingStatus defines the observed status details of Knative Eventing.
type KnativeEventingStatus struct {
	Ready        string `json:"ready,omitempty"`
	ErrorMessage string `json:"errorMessage,omitempty"`
	Version      string `json:"version,omitempty"`
}

// ServerlessStatus defines the observed status details of Open Shift serverless.
type ServerlessStatus struct {
	Ready          string               `json:"ready,omitempty"`
	ErrorMessage   string               `json:"errorMessage,omitempty"`
	Version        string               `json:"version,omitempty"`
	KnativeServing KnativeServingStatus `json:"knativeServing,omitempty"`
}

// KnativeServingStatus defines the observed status details of Knative Serving.
type KnativeServingStatus struct {
	Ready        string `json:"ready,omitempty"`
	ErrorMessage string `json:"errorMessage,omitempty"`
	Version      string `json:"version,omitempty"`
}

// CliStatus defines the observed status details of the Kabanero CLI.
type CliStatus struct {
	Ready        string   `json:"ready,omitempty"`
	ErrorMessage string   `json:"errorMessage,omitempty"`
	Hostnames    []string `json:"hostnames,omitempty"`
}

// KabaneroLandingPageStatus defines the observed status details of the Kabanero landing page.
type KabaneroLandingPageStatus struct {
	Ready        string `json:"ready,omitempty"`
	ErrorMessage string `json:"errorMessage,omitempty"`
	Version      string `json:"version,omitempty"`
}

// AppsodyStatus defines the observed status details of Appsody.
type AppsodyStatus struct {
	Ready        string `json:"ready,omitempty"`
	ErrorMessage string `json:"errorMessage,omitempty"`
	Version      string `json:"version,omitempty"`
}

// KappnavStatus defines the observed status details of Kubernetes Application Navigator.
type KappnavStatus struct {
	Ready        string   `json:"ready,omitempty"`
	ErrorMessage string   `json:"errorMessage,omitempty"`
	UiLocations  []string `json:"uiLocations,omitempty"`
	ApiLocations []string `json:"apiLocations,omitempty"`
}

// CheStatus defines the observed status details of Che.
type CheStatus struct {
	Ready               string                    `json:"ready,omitempty"`
	ErrorMessage        string                    `json:"errorMessage,omitempty"`
	CheOperator         CheOperatorStatus         `json:"cheOperator,omitempty"`
	KabaneroChe         KabaneroCheStatus         `json:"kabaneroChe,omitempty"`
	KabaneroCheInstance KabaneroCheInstanceStatus `json:"kabaneroCheInstance,omitempty"`
}

// CheOperatorStatus defines the observed status details of the Che operator.
type CheOperatorStatus struct {
	Version string `json:"version,omitempty"`
}

// KabaneroCheStatus defines the observed status details of Kabanero Che.
type KabaneroCheStatus struct {
	Version string `json:"version,omitempty"`
}

// KabaneroCheInstanceStatus defines the observed status details of Che instance.
type KabaneroCheInstanceStatus struct {
	CheImage                string `json:"cheImage,omitempty"`
	CheImageTag             string `json:"cheImageTag,omitempty"`
	CheWorkspaceClusterRole string `json:"cheWorkspaceClusterRole,omitempty"`
}

// WebhookStatus defines the observed status details of the Kabanero webhook.
type WebhookStatus struct {
	Ready        string   `json:"ready,omitempty"`
	ErrorMessage string   `json:"errorMessage,omitempty"`
	Hostnames    []string `json:"hostnames,omitempty"`
}

// AdmissionControllerWebhookStatus defines the observed status details of the Kabanero mutating and validating admission webhooks.
type AdmissionControllerWebhookStatus struct {
	Ready        string   `json:"ready,omitempty"`
	ErrorMessage string   `json:"errorMessage,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Kabanero is the Schema for the kabaneros API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type Kabanero struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KabaneroSpec   `json:"spec,omitempty"`
	Status KabaneroStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KabaneroList contains a list of Kabanero
type KabaneroList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Kabanero `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Kabanero{}, &KabaneroList{})
}
