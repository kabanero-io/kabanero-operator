package v1alpha2

// The component interfaces are an abstraction on a component is capable of
// running multiple versions of itself simultaneously, with each version
// potentially using different version of Kabanero pipelines.
//
// Currently both the Kabanero CR and Stack CR support these interfaces,
// although in practice the Kabanero CR only supports a single version.

// The status for a single version of a versioned component.
type ComponentStatusVersion interface {
	// The descriptive version name (ie "0.1.2")
	GetVersion() (string)
	// A list of pipelines that are currently being used by this version
	// of the component.
	GetPipelines() ([]PipelineStatus)
}

// Aggregated status for all versions of a versioned component.
type ComponentStatus interface {
	// A list of versions that are currently active.
	GetVersions() ([]ComponentStatusVersion)
}

// The specification of a single version of a multi-versioned component.
type ComponentSpecVersion interface {
	// The descriptive version name (ie "0.1.2")
	GetVersion() (string)
	// A list of pipelines that should be activated for this version of
	// the component.
	GetPipelines() ([]PipelineSpec)
}

// Aggregated specification for all versions of a versioned component.
type ComponentSpec interface {
	// A list of versions that should be activated.
	GetVersions() ([]ComponentSpecVersion)
}
