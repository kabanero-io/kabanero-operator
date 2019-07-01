## Kabanero Instance

A Kabanero instance is described by a resource definition with `kind: Kabanero`:

```
apiVersion: kabanero.io/v1alpha1
kind: Kabanero
metadata:
  name: kabanero
spec:
  ...
```

Creation of a Kabanero Instance has multiple impacts: 
* May cause cluster level configuration, such as KNative Eventing being enabled on the cluster
* May cause deployment of instance specific resources, such as dashboard user interfaces, API endpoints, etc. 

## Collections

A collection is scoped to a namespace. When a collection is applied, there may be a number of Kubernetes resources which come with the collection, and these are applied into the same namespace as the collection resource. 

### Example

The following `Collection` resource is assigned to the namespace `mynamespace`: 
```
apiVersion: kabanero.io/v1alpha1
kind: Collection
metadata:
  name: java-microprofile
  namespace: mynamespace
spec:
  version: 1.0.0
```

This collection will create a number of Tekton pipelines in the same namespace: 
```
apiVersion: 
kind: Pipeline
metadata:
  name: mypipeline
  namespace: mynamespace
spec:
  ...
```

For further details see [collections](collections.md)