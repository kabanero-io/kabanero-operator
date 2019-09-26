# Explanation of samples

* `default.yaml` shows the basic configuration that most people use.  It is referenced in the [Kabanero installation script](https://github.com/kabanero-io/kabanero-foundation/tree/master/scripts).  A `version` is provided so the components that make up Kabanero will not be upgraded as the operator discovers new releases.

* `full.yaml` shows every possible configuration option.  Many of these options are only used by developers (for example, over-riding the container image used by the various Kabanero components).  This yaml is not suitable for use as-is, it must be customized.

* `simple.yaml` shows the simplest possible configuration.  Since no `version` is provided, the components that make up Kabanero will be upgraded as the operator discovers new versions are available in a named release.  

* `override_software_versions.yaml` shows several examples of `kind: Kabanero` that over-ride the `version` of one or more Kabanero components.  This yaml is not suitable for use as-is, since it contains several `kind: Kabanero` instances.

* `collection.yaml` shows how to activate a specific Kabanero collection from the collection repository configured in `kind: Kabanero`.  Generally this is not required since the default behavior is to activate all collections.