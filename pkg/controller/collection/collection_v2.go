package collection


// Convenience function which iterates over the complex Collections structure
//func (c *CollectionV1Index) ListCollectionsV2() []IndexedCollectionV2 {
//	all := make([]IndexedCollectionV2, 0)
//	for _, v := range c.CollectionsV2 {
//		for _, colRef := range v {
//			all = append(all, colRef)
//		}
//	}
//
//	return all
//}

type IndexedCollectionV2 struct {
	DefaultDashboard   string                   `yaml:"default-dashboard,omitempty"`
	DefaultImage       string                   `yaml:"default-image,omitempty"`
	DefaultPipeline    string                   `yaml:"default-pipeline,omitempty"`
	DefaultTemplate    string                   `yaml:"default-template,omitempty"`
	Description        string                   `yaml:"description,omitempty"`
	Id                 string                   `yaml:"id,omitempty"`
	Images             []IndexedImagesV2        `yaml:"images,omitempty"`
	License            string                   `yaml:"license,omitempty"`
	Maintainers        []IndexedMaintainersV2   `yaml:"maintainers,omitempty"`
	Name               string                   `yaml:"name,omitempty"`
	Pipelines          []IndexedPipelinesV2     `yaml:"pipelines,omitempty"`
	Templates          []IndexedTemplatesV2     `yaml:"templates,omitempty"`
	Version            string                   `yaml:"version,omitempty"`
}

type IndexedImagesV2 struct {
	Id                 string                   `yaml:"id,omitempty"`
	Image              string                   `yaml:"image,omitempty"`
}

type IndexedMaintainersV2 struct {
	Email              string                   `yaml:"email,omitempty"`
	GithubId           string                   `yaml:"github-id,omitempty"`
	Name               string                   `yaml:"name,omitempty"`
}

type IndexedPipelinesV2 struct {
	Id                 string                   `yaml:"id,omitempty"`
	Sha256             string                   `yaml:"sha256,omitempty"`
	Url                string                   `yaml:"url,omitempty"`
}

type IndexedTemplatesV2 struct {
	Id                 string                   `yaml:"id,omitempty"`
	Url                string                   `yaml:"url,omitempty"`
}

type PipelineManifestV2 struct {
	Contents           []PipelineFilesV2        `yaml:"contents,omitempty"`
}

type PipelineFilesV2 struct {
	File               string                   `yaml:"file,omitempty"`
	Sha256             string                   `yaml:"sha256,omitempty"`
}