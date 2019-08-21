package collection

type CollectionV1Index struct {
	ApiVersion  string                           `yaml:"apiVersion,omitempty"`
	
	// V1 Collections
	Generated   string                           `yaml:"generated,omitempty"`
	Collections map[string][]IndexedCollectionV1 `yaml:"projects,omitempty"`
	
	// V2 Collections
	CollectionsV2 []IndexedCollectionV2 `yaml:"stacks,omitempty"`
}
