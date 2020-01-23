package v1alpha2

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.
// NOTE: The +listType=set marker is required by OpenAPI generation for list types.

// +kubebuilder:subresource:status

// KabaneroSpec defines the desired state of Kabanero
// +k8s:openapi-gen=true
type KabaneroSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html

	Version string `json:"version,omitempty"`

	// +listType=set
	TargetNamespaces []string `json:"targetNamespaces,omitempty"`

	Github GithubConfig `json:"github,omitempty"`

	Stacks InstanceStackConfig `json:"stacks,omitempty"`

	// +listType=set
	Triggers []TriggerSpec `json:"triggers,omitempty"`

	CliServices KabaneroCliServicesCustomizationSpec `json:"cliServices,omitempty"`

	Landing KabaneroLandingCustomizationSpec `json:"landing,omitempty"`

	Che CheCustomizationSpec `json:"che,omitempty"`

	Events EventsCustomizationSpec `json:"events,omitempty"`

	StackController StackControllerSpec `json:"stackController,omitempty"`

	AdmissionControllerWebhook AdmissionControllerWebhookCustomizationSpec `json:"admissionControllerWebhook,omitempty"`
}

// InstanceStackConfig defines the customization entries for a set of stacks.
type InstanceStackConfig struct {
	// +listType=set
	Repositories []RepositoryConfig `json:"repositories,omitempty"`

	// +listType=set
	Pipelines []PipelineSpec `json:"pipelines,omitempty"`
}

// PipelineSpec defines the sets of default pipelines for the stacks.
type PipelineSpec struct {
	Id string `json:"id,omitempty"`
	Sha256 string `json:"sha256,omitempty"`
	Https HttpsProtocolFile `json:"https,omitempty"`
}

// HttpsProtocolFile defines how to retrieve a file over https
type HttpsProtocolFile struct {
	Url                  string `json:"url,omitempty"`
	SkipCertVerification bool   `json:"skipCertVerification,omitempty"`
}

// TriggerSpec defines the sets of default triggers for the stacks
type TriggerSpec struct {
	Id string `json:"id,omitempty"`
	Sha256 string `json:"sha256,omitempty"`
	Https HttpsProtocolFile `json:"https,omitempty"`
}

// GithubConfig represents the Github information (public or GHE) where
// the organization and teams managing the stacks live.  Members
// of the specified team in the specified organization will have admin
// authority in the Kabanero CLI.
type GithubConfig struct {
	Organization string `json:"organization,omitempty"`
	// +listType=set
	Teams  []string `json:"teams,omitempty"`
	ApiUrl string   `json:"apiUrl,omitempty"`
}

// RepositoryConfig defines customization entries for a stack.
type RepositoryConfig struct {
	Name                       string `json:"name,omitempty"`
	// +listType=set
	Pipelines                  []PipelineSpec `json:"pipelines,omitempty"`
	Https                      HttpsProtocolFile `json:"https,omitempty"`
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

type EventsCustomizationSpec struct {
	Enable     bool   `json:"enable,omitempty"`
	Version    string `json:"version,omitempty"`
	Image      string `json:"image,omitempty"`
	Repository string `json:"repository,omitempty"`
	Tag        string `json:"tag,omitempty"`
}

// StackControllerSpec defines customization entried for the Kabanero stack controller.
type StackControllerSpec struct {
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

	// Events instance status
	Events *EventsStatus `json:"events,omitempty"`

	// Kabanero stack controller readiness status.
	StackController StackControllerStatus `json:"stackController,omitempty"`

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
	Ready        string `json:"ready,omitempty"`
	ErrorMessage string `json:"errorMessage,omitempty"`
	// +listType=set
	Hostnames []string `json:"hostnames,omitempty"`
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
	Ready        string `json:"ready,omitempty"`
	ErrorMessage string `json:"errorMessage,omitempty"`
	// +listType=set
	UiLocations []string `json:"uiLocations,omitempty"`
	// +listType=set
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

// EventsStatus defines the observed status details of the Kabanero events.
type EventsStatus struct {
	Ready        string `json:"ready,omitempty"`
	ErrorMessage string `json:"errorMessage,omitempty"`
	// +listType=set
	Hostnames []string `json:"hostnames,omitempty"`
}

// StackControllerStatus defines the observed status details of the Kabanero stack controller.
type StackControllerStatus struct {
	Ready        string `json:"ready,omitempty"`
	ErrorMessage string `json:"errorMessage,omitempty"`
	Version      string `json:"version,omitempty"`
}

// AdmissionControllerWebhookStatus defines the observed status details of the Kabanero mutating and validating admission webhooks.
type AdmissionControllerWebhookStatus struct {
	Ready        string `json:"ready,omitempty"`
	ErrorMessage string `json:"errorMessage,omitempty"`
}

// Kabanero is the Schema for the kabaneros API
// Note that kubebuilder and operator-sdk currently disagree about what the
// plural of this type should be.  The +kubebuilder:resource marker sets the
// plural to what operator-sdk expects.

// +k8s:openapi-gen=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",description="CreationTimestamp is a timestamp representing the server time when this object was created. It is not guaranteed to be set in happens-before order across separate operations."
// +kubebuilder:printcolumn:name="Version",type="string",JSONPath=".status.kabaneroInstance.version",description="Kabanero operator instance version."
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.kabaneroInstance.ready",description="Kabanero operator instance readiness status. The status is directly correlated to the availability of the operator's resources dependencies."
// +kubebuilder:resource:path=kabaneros,scope=Namespaced
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
	// +listType=set
	Items []Kabanero `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Kabanero{}, &KabaneroList{})
}
