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

	CodereadyWorkspaces CRWCustomizationSpec `json:"codeReadyWorkspaces,omitempty"`

	Events EventsCustomizationSpec `json:"events,omitempty"`

	CollectionController CollectionControllerSpec `json:"collectionController,omitempty"`

	StackController StackControllerSpec `json:"stackController,omitempty"`

	AdmissionControllerWebhook AdmissionControllerWebhookCustomizationSpec `json:"admissionControllerWebhook,omitempty"`

	Sso SsoCustomizationSpec `json:"sso,omitempty"`
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
	Id     string            `json:"id,omitempty"`
	Sha256 string            `json:"sha256,omitempty"`
	Https  HttpsProtocolFile `json:"https,omitempty"`
}

// HttpsProtocolFile defines how to retrieve a file over https
type HttpsProtocolFile struct {
	Url                  string `json:"url,omitempty"`
	SkipCertVerification bool   `json:"skipCertVerification,omitempty"`
}

// TriggerSpec defines the sets of default triggers for the stacks
type TriggerSpec struct {
	Id     string            `json:"id,omitempty"`
	Sha256 string            `json:"sha256,omitempty"`
	Https  HttpsProtocolFile `json:"https,omitempty"`
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
	Name string `json:"name,omitempty"`
	// +listType=set
	Pipelines  []PipelineSpec    `json:"pipelines,omitempty"`
	Https      HttpsProtocolFile `json:"https,omitempty"`
	GitRelease GitReleaseSpec    `json:"gitRelease,omitempty"`
}

// GitReleaseSpec defines customization entries for a Git release.
type GitReleaseSpec struct {
	Hostname     string `json:"hostname,omitempty"`
	Organization string `json:"organization,omitempty"`
	Project      string `json:"project,omitempty"`
	Release      string `json:"release,omitempty"`
	AssetName    string `json:"assetName,omitempty"`
	SkipCertVerification bool `json:"skipCertVerification,omitempty"`
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

// CRWCustomizationSpec defines customization entries for codeready-workspaces.
type CRWCustomizationSpec struct {
	Enable   *bool           `json:"enable,omitempty"`
	Operator CRWOperatorSpec `json:"operator,omitempty"`
}

// CRWOperatorSpec defines customization entries for the codeready-workspaces operator.
type CRWOperatorSpec struct {
	CustomResourceInstance CRWOperatorCRInstanceSpec `json:"customResourceInstance,omitempty"`
}

// CRWOperatorCustomResourceSpec defines custom resource customization entries for the codeready-workspaces operator.
type CRWOperatorCRInstanceSpec struct {
	DevFileRegistryImage    CWRCustomResourceDevFileRegImage `json:"devFileRegistryImage,omitempty"`
	CheWorkspaceClusterRole string                           `json:"cheWorkspaceClusterRole,omitempty"`
	OpenShiftOAuth          *bool                            `json:"openShiftOAuth,omitempty"`
	SelfSignedCert          *bool                            `json:"selfSignedCert,omitempty"`
	TLSSupport              *bool                            `json:"tlsSupport,omitempty"`
}

// CWRCustomResourceDevFileRegImage defines DevFileRegistryImage custom resource customization for the codeready-workspaces operator.
type CWRCustomResourceDevFileRegImage struct {
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

// CollectionControllerSpec defines customization entried for the Kabanero collection controller.
type CollectionControllerSpec struct {
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

type SsoCustomizationSpec struct {
	Enable          bool   `json:"enable,omitempty"`
	Provider        string `json:"provider,omitempty"`
	AdminSecretName string `json:"adminSecretName,omitempty"`
}

// KabaneroStatus defines the observed state of the Kabanero instance.
// +k8s:openapi-gen=true
type KabaneroStatus struct {
	// Kabanero operator instance readiness status. The status is directly correlated to the availability of resources dependencies.
	KabaneroInstance KabaneroInstanceStatus `json:"kabaneroInstance,omitempty"`

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

	// Codeready-workspaces instance readiness status.
	CodereadyWorkspaces *CRWStatus `json:"codereadyWorkspaces,omitempty"`

	// Events instance status
	Events *EventsStatus `json:"events,omitempty"`

	// Kabanero collection controller readiness status.
	CollectionController CollectionControllerStatus `json:"collectionController,omitempty"`

	// Kabanero stack controller readiness status.
	StackController StackControllerStatus `json:"stackController,omitempty"`

	// Admission webhook instance status
	AdmissionControllerWebhook AdmissionControllerWebhookStatus `json:"admissionControllerWebhook,omitempty"`

	// SSO server status
	Sso SsoStatus `json:"sso,omitempty"`
}

// KabaneroInstanceStatus defines the observed status details of Kabanero operator instance
type KabaneroInstanceStatus struct {
	Ready   string `json:"ready,omitempty"`
	Message string `json:"message,omitempty"`
	Version string `json:"version,omitempty"`
}

// TektonStatus defines the observed status details of Tekton.
type TektonStatus struct {
	Ready   string `json:"ready,omitempty"`
	Message string `json:"message,omitempty"`
	Version string `json:"version,omitempty"`
}

// ServerlessStatus defines the observed status details of Open Shift serverless.
type ServerlessStatus struct {
	Ready          string               `json:"ready,omitempty"`
	Message        string               `json:"message,omitempty"`
	Version        string               `json:"version,omitempty"`
	KnativeServing KnativeServingStatus `json:"knativeServing,omitempty"`
}

// KnativeServingStatus defines the observed status details of Knative Serving.
type KnativeServingStatus struct {
	Ready   string `json:"ready,omitempty"`
	Message string `json:"message,omitempty"`
	Version string `json:"version,omitempty"`
}

// CliStatus defines the observed status details of the Kabanero CLI.
type CliStatus struct {
	Ready   string `json:"ready,omitempty"`
	Message string `json:"message,omitempty"`
	// +listType=set
	Hostnames []string `json:"hostnames,omitempty"`
}

// KabaneroLandingPageStatus defines the observed status details of the Kabanero landing page.
type KabaneroLandingPageStatus struct {
	Ready   string `json:"ready,omitempty"`
	Message string `json:"message,omitempty"`
	Version string `json:"version,omitempty"`
}

// AppsodyStatus defines the observed status details of Appsody.
type AppsodyStatus struct {
	Ready   string `json:"ready,omitempty"`
	Message string `json:"message,omitempty"`
	Version string `json:"version,omitempty"`
}

// KappnavStatus defines the observed status details of Kubernetes Application Navigator.
type KappnavStatus struct {
	Ready   string `json:"ready,omitempty"`
	Message string `json:"message,omitempty"`
	// +listType=set
	UiLocations []string `json:"uiLocations,omitempty"`
	// +listType=set
	ApiLocations []string `json:"apiLocations,omitempty"`
}

// CRWStatus defines the observed status details of codeready-workspaces.
type CRWStatus struct {
	Ready    string            `json:"ready,omitempty"`
	Message  string            `json:"message,omitempty"`
	Operator CRWOperatorStatus `json:"operator,omitempty"`
}

// CRWOperatorStatus defines the observed status details of the codeready-workspaces operator.
type CRWOperatorStatus struct {
	Version  string            `json:"version,omitempty"`
	Instance CRWInstanceStatus `json:"instance,omitempty"`
}

// CRWInstanceStatus defines the observed status details of the codeready-workspaces operator custom resource.
type CRWInstanceStatus struct {
	DevfileRegistryImage    string `json:"devfileRegistryImage"`
	CheWorkspaceClusterRole string `json:"cheWorkspaceClusterRole"`
	OpenShiftOAuth          bool   `json:"openShiftOAuth"`
	SelfSignedCert          bool   `json:"selfSignedCert"`
	TLSSupport              bool   `json:"tlsSupport"`
}

// EventsStatus defines the observed status details of the Kabanero events.
type EventsStatus struct {
	Ready   string `json:"ready,omitempty"`
	Message string `json:"message,omitempty"`
	// +listType=set
	Hostnames []string `json:"hostnames,omitempty"`
}

// CollectionControllerStatus defines the observed status details of the Kabanero collection controller.
type CollectionControllerStatus struct {
	Ready   string `json:"ready,omitempty"`
	Message string `json:"message,omitempty"`
	Version string `json:"version,omitempty"`
}

// StackControllerStatus defines the observed status details of the Kabanero stack controller.
type StackControllerStatus struct {
	Ready   string `json:"ready,omitempty"`
	Message string `json:"message,omitempty"`
	Version string `json:"version,omitempty"`
}

// AdmissionControllerWebhookStatus defines the observed status details of the Kabanero mutating and validating admission webhooks.
type AdmissionControllerWebhookStatus struct {
	Ready   string `json:"ready,omitempty"`
	Message string `json:"message,omitempty"`
}

// Status of the SSO server
type SsoStatus struct {
	Configured string `json:"configured,omitempty"`
	Ready      string `json:"ready,omitempty"`
	Message    string `json:"message,omitempty"`
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
// +kubebuilder:storageversion
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
