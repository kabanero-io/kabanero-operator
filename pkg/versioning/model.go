package versioning

import (
	"github.com/kabanero-io/kabanero-operator/pkg/assets/config"
	"gopkg.in/yaml.v2"
	"net/http"
)

var Data = func() VersionDocument {
	f, err := config.Open("versions.yaml")
	if err != nil {
		panic(err)
	}

	dec := yaml.NewDecoder(f)
	var versionData VersionDocument
	err = dec.Decode(&versionData)
	if err != nil {
		panic(err)
	}

	//Update the pointer between Kabanero's and the document
	for i, k := range versionData.KabaneroRevisions {
		k.Document = &versionData
		versionData.KabaneroRevisions[i] = k
	}

	return versionData
}()

// The top level resource within the versioning model
type VersionDocument struct {
	// The version of Kabanero to use if otherwise unspecified
	DefaultKabaneroRevision string `yaml:"default,omitempty"`

	// The versions of Kabanero known to this operator
	KabaneroRevisions []KabaneroRevision `yaml:"kabanero,omitempty"`

	// The versions of related software known to this operator
	RelatedSoftwareRevisions map[string][]SoftwareRevision `yaml:"related-software,omitempty"`
}

func (doc VersionDocument) KabaneroRevision(KabaneroRevision string) *KabaneroRevision {
	for _, k := range doc.KabaneroRevisions {
		if k.Version == KabaneroRevision {
			return &k
		}
	}

	return nil
}

// A specific version of Kabanero, contains the association to the other related software versions
// which match this Kabanero version
type KabaneroRevision struct {
	Version string `yaml:"version,omitempty"`

	// The versions associated with this Kabanero Version
	RelatedVersions map[string]string `yaml:"related-versions,omitempty"`

	Document *VersionDocument `yaml:"-"`
}

func (KabaneroRevision KabaneroRevision) SoftwareComponent(softwareComponent string) *SoftwareRevision {
	if relatedVersion, ok := KabaneroRevision.RelatedVersions[softwareComponent]; ok {
		related := KabaneroRevision.Document.RelatedSoftwareRevisions
		relatedVersions := related[softwareComponent]
		for _, revision := range relatedVersions {
			if revision.Version == relatedVersion {
				return &revision
			}
		}
	}

	return nil
}

// Contains version specific data for software which is orchestrated as part of Kabanero
type SoftwareRevision struct {
	// The version of this piece of software that the orchestrations and identifiers apply to
	Version string `yaml:"version,omitempty"`

	// The path to orchestration sources
	OrchestrationPath string `yaml:"orchestrations,omitempty"`

	// Identifiers contains key/value pairs which are specific to this revision of the sofware
	Identifiers map[string]interface{} `yaml:"identifiers,omitempty"`
}

// Opens the embedded orchestration file using the internal OrchestrationPath + provided path
func (rev SoftwareRevision) OpenOrchestration(path string) (http.File, error) {
	f, err := config.Open(rev.OrchestrationPath + "/" + path)
	return f, err
}
