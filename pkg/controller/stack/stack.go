package stack

import kabanerov1alpha2 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha2"

// Stack holds stack specific data.
type Stack struct {
	DefaultDashboard string        `yaml:"default-dashboard,omitempty"`
	DefaultImage     string        `yaml:"default-image,omitempty"`
	DefaultPipeline  string        `yaml:"default-pipeline,omitempty"`
	DefaultTemplate  string        `yaml:"default-template,omitempty"`
	Description      string        `yaml:"description,omitempty"`
	Id               string        `yaml:"id,omitempty"`
	Image            string        `yaml:"image,omitempty"`
	Images           []Images      `yaml:"images,omitempty"`
	License          string        `yaml:"license,omitempty"`
	Maintainers      []Maintainers `yaml:"maintainers,omitempty"`
	Name             string        `yaml:"name,omitempty"`
	Pipelines        []Pipelines   `yaml:"pipelines,omitempty"`
	Templates        []Templates   `yaml:"templates,omitempty"`
	Version          string        `yaml:"version,omitempty"`
}

// Images holds a stack image data.
type Images struct {
	Id    string `yaml:"id,omitempty"`
	Image string `yaml:"image,omitempty"`
}

// Maintainers holds stack maintainer information.
type Maintainers struct {
	Email    string `yaml:"email,omitempty"`
	GithubId string `yaml:"github-id,omitempty"`
	Name     string `yaml:"name,omitempty"`
}

// Pipelines holds a stack's associated pipeline data.
type Pipelines struct {
	Id                   string                          `yaml:"id,omitempty"`
	Sha256               string                          `yaml:"sha256,omitempty"`
	Url                  string                          `yaml:"url,omitempty"`
	GitRelease           kabanerov1alpha2.GitReleaseSpec `yaml:"gitRelease,omitempty"`
	SkipCertVerification bool                            `yaml:"skipCertVerification,omitempty"`
}

// Templates holds the stack's associated template data.
type Templates struct {
	Id  string `yaml:"id,omitempty"`
	Url string `yaml:"url,omitempty"`
}

// PipelineManifest holds the stack's associated pipelime manifests.
type PipelineManifest struct {
	Contents []PipelineFiles `yaml:"contents,omitempty"`
}

// PipelineFiles holds the stack's associated pipeline files.
type PipelineFiles struct {
	File   string `yaml:"file,omitempty"`
	Sha256 string `yaml:"sha256,omitempty"`
}
