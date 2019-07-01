package collection

type CollectionV1Index struct {
	ApiVersion  string                           `yaml:"apiVersion,omitempty"`
	Generated   string                           `yaml:"generated,omitempty"`
	Collections map[string][]IndexedCollectionV1 `yaml:"projects,omitempty"`
}

type IndexedCollectionV1 struct {
	Created        string   `yaml:"created,omitempty"`
	Description    string   `yaml:"description,omitempty"`
	Icon           string   `yaml:"icon,omitempty"`
	Keywords       []string `yaml:"keywords,omitempty"`
	Name           string   `yaml:"name,omitempty"`
	Maintainers    []string `yaml:"maintainers,omitempty"`
	Urls           []string `yaml:"urls,omitempty"`
	CollectionUrls []string `yaml:"collectionUrls,omitempty"`
}

type CollectionV1 struct {
	Manifest CollectionV1Manifest
}

type CollectionV1Manifest struct {
	APIVersion  string          `json:"apiVersion,omitempty"`
	Name        string          `json:"name,omitempty"`
	Version     string          `json:"version,omitempty"`
	Description string          `json:"description,omitempty"`
	Icon        string          `json:"icon,omitempty"`
	Keywords    []string        `json:"keywords,omitempty"`
	Assets      []AssetManifest `json:"assets,omitempty"`
}

type AssetManifest struct {
	Name   string `json:"name,omitempty"`
	Type   string `json:"type,omitempty"`
	Url    string `json:"url,omitempty"`
	Digest string `json:"digest,omitempty"`
}
