package stack

// Index holds data pertaining to an index referencing a set of stacks.
type Index struct {
	// API Version.
	APIVersion string `yaml:"apiVersion,omitempty"`

	// Source URL of this index
	URL string `yaml:"url,omitempty"`

	// Holds version 2 stack's data.
	Stacks []Stack `yaml:"stacks,omitempty"`
}
