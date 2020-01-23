package collection

// Index holds data pertaining to an index referencing a set of collections.
type Index struct {
	// API Version.
	APIVersion string `yaml:"apiVersion,omitempty"`

	// Holds version 2 collection's data.
	Collections []Collection `yaml:"stacks,omitempty"`

	// Holds version 2 collection's data.
	Triggers []Trigger `yaml:"triggers,omitempty"`

	// Source URL of this index
	URL string `yaml:"url,omitempty"`
}

// Trigger holds Trigger information.
type Trigger struct {
	Id     string `yaml:"id,omitempty"`
	Url    string `yaml:"url,omitempty"`
	Sha256 string `yaml:"sha256,omitempty"`
}