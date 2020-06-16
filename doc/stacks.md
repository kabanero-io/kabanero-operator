# Stacks

Stack contents are maintained in a Kabanero repository. A Kabanero instance refers to one or more stacks repositories and can additionally specify whether *featured* stacks are to be activated through the `enableFeatured` attribute. 

*TODO:* consider moving the enableFeatured attribute to the repository. 

## Stack Activation

Stacks are said to be *activated* based upon the presence of a Stack resource kind which references a stack by name and version. When searching for a stack by name, the repository ordering found in the Kabanero instance will be respected.

## Stack Upgrade

Only one version of a stack can be active in a particular namespace at a time. The stack resource will reference the currently activated version. By updating the 'version' attribute of the stack spec, a new version can be activated. 

## Featured Stacks

The maintainer of a stacks repository can choose to flag certain stacks are being "featured". When a stacks repository is added to a Kabanero instance and the installation of featured stacks is enabled, the featured stacks are identified and activated. 

## Removal of Stack Repositories

A stack repository can be removed from a Kabanero instance by updating the stack repository list, for example: 

```
  stacks:
    # A list of those repositories which are searched for stacks
    repositories: 
    - name: release-0.8
      https:
        url: https://github.com/kabanero-io/kabanero-stack-hub/releases/download/0.8.0/kabanero-stack-hub-index.yaml
    - name: incubator
      https:
        url: https://github.com/kabanero-io/kabanero-stack-hub/releases/download/0.9.0-rc.1/kabanero-stack-hub-index.yaml
    pipelines:
    - id: default
      sha256: 3f3e440b3eed24273fd43c40208fdd95de6eadeb82b7bb461f52e1e5da7e239d
      https:
        url: https://github.com/kabanero-io/kabanero-pipelines/releases/download/0.8.0/default-kabanero-pipelines.tar.gz
```

could be changed to:
```
  stacks:
    # A list of those repositories which are searched for stacks
    repositories: 
    - name: release-0.8
      https:
        url: https://github.com/kabanero-io/kabanero-stack-hub/releases/download/0.8.0/kabanero-stack-hub-index.yaml
    pipelines:
    - id: default
      sha256: 3f3e440b3eed24273fd43c40208fdd95de6eadeb82b7bb461f52e1e5da7e239d
      https:
        url: https://github.com/kabanero-io/kabanero-pipelines/releases/download/0.8.0/default-kabanero-pipelines.tar.gz
```

When a stack repository is removed from the list, no action is taken unless all of the referenced stack resources have also been removed.
