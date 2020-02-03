# Collections

Collection contents are maintained in a Kabanero repository. A Kabanero instance refers to one or more collections repositories and can additionally specify whether *featured* collections are to be activated through the `enableFeatured` attribute. 

*TODO:* consider moving the enableFeatured attribute to the repository. 

## Collection Activation

Collections are said to be *activated* based upon the presence of a Collection resource kind which references a collection by name and version. When searching for a collection by name, the repository ordering found in the Kabanero instance will be respected.

## Collection Upgrade

Only one version of a collection can be active in a particular namespace at a time. The collection resource will reference the currently activated version. By updating the 'version' attribute of the collection spec, a new version can be activated. 

## Featured Collections

The maintainer of a collections repository can choose to flag certain collections are being "featured". When a collections repository is added to a Kabanero instance and the installation of featured collections is enabled, the featured collections are identified and activated. 

## Removal of Collection Repositories

A collection repository can be removed from a Kabanero instance by updating the collection repository list, for example: 

```
  collections: 
    repositories: 
    - name: experimental
      url: https://github.com/kabanero-io/kabanero-collection/blob/master/experimental
    - name: my-experimental
      url: https://myrepo.com/experimental
```

could be changed to:
```
  collections: 
    repositories: 
    - name: experimental
      url: https://myrepo.com/experimental
```

When a collection repository is removed from the list, no action is taken unless all of the referenced collection resources have also been removed.
