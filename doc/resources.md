## Kabanero Instance

A Kabanero instance is described by a resource definition with `kind: Kabanero`:

```
apiVersion: kabanero.io/v1alpha2
kind: Kabanero
metadata:
  name: kabanero
spec:
  ...
```

Creation of a Kabanero Instance has multiple impacts: 
* May cause cluster level configuration, such as KNative Serving being enabled on the cluster
* May cause deployment of instance specific resources, such as dashboard user interfaces, API endpoints, etc. 

## Stacks

A stack is scoped to a namespace. When a stack is applied, there may be a number of Kubernetes resources which come with the stack, and these are applied into the same namespace as the stack resource. 

### Example

The following `Stack` resource is assigned to the namespace `mynamespace`: 
```
apiVersion: kabanero.io/v1alpha2
kind: Stack
metadata:
  name: java-microprofile
  namespace: mynamespace
spec:
  version: 1.0.0
```

This stack will create a number of Tekton pipelines in the same namespace: 
```
apiVersion: 
kind: Pipeline
metadata:
  name: mypipeline
  namespace: mynamespace
spec:
  ...
```

For further details see [stacks](stacks.md)