package collection

// Index holds data pertaining to an index referencing a set of collections.
type Index struct {
	// API Version.
	APIVersion string `yaml:"apiVersion,omitempty"`

	// Source URL of this index
	URL string `yaml:"url,omitempty"`

	// Holds version 2 collection's data.
	Collections []Collection `yaml:"stacks,omitempty"`
}
