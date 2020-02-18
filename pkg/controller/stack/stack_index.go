package stack

// Index holds data pertaining to an index referencing a set of stacks.
type Index struct {
	// API Version.
	APIVersion string `yaml:"apiVersion,omitempty"`

	// Holds version 2 stack's data.
	Stacks []Stack `yaml:"stacks,omitempty"`

	// Holds version 2 stack's data.
	Triggers []Trigger `yaml:"triggers,omitempty"`
}

// Trigger holds Trigger information.
type Trigger struct {
	Id     string `yaml:"id,omitempty"`
	Url    string `yaml:"url,omitempty"`
	Sha256 string `yaml:"sha256,omitempty"`
}
