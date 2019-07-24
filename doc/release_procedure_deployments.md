# Setup Release (sometime after last release)

Before starting, create the release deployment staging directory. This is referenced in the next steps: 

```
mkdir deploy/releases/<next release identifier>
cd deploy/releases/<next release identifier>
```

## Update Prereqs

### KNative Eventing

Determine the latest stable tag in the repository https://github.com/openshift-knative/knative-eventing-operator

Update contrib/get_knative_eventing_operator.sh then run it:
```
../../../contrib/get_knative_eventing_operator.sh
```

In the top of the generated `knative-eventing.yaml` file, add a create namespace resource:
``` 
apiVersion: v1
kind: Namespace
metadata:
  name: knative-eventing
---
```

For each namespaced object in knative-eventing.yaml (i.e. not for crds or cluster scope RBAC objects), set the namespace: 
```
namespace: knative-eventing
```


### KNative Serving

Repeat the knative-eventing procedure with knative-serving

### Tekton Pipelines

Run tekton script and add the namespace to the yaml file: 

```
apiVersion: v1
kind: Namespace
metadata:
  name: openshift-pipelines-operator
---
```

## Create Kabanero Operator Yaml

```
cat ../../operator.yaml > kabanero-operator.yaml; echo "---" >> kabanero-operator.yaml
cat ../../role.yaml >> kabanero-operator.yaml; echo "---" >> kabanero-operator.yaml
cat ../../role_binding.yaml >> kabanero-operator.yaml; echo "---" >> kabanero-operator.yaml
cat ../../service_account.yaml >> kabanero-operator.yaml; echo "---" >> kabanero-operator.yaml
cat ../../crds/kabanero_v1alpha1_collection_crd.yaml >> kabanero-operator.yaml; echo "---" >> kabanero-operator.yaml
cat ../../crds/kabanero_v1alpha1_kabanero_crd.yaml >> kabanero-operator.yaml; echo "---" >> kabanero-operator.yaml
```

Add the namespace to the kabanero-operator.yaml file:
```
apiVersion: v1
kind: Namespace
metadata:
  name: kabanero
```

Set the namespace for all namespace scoped objects to:
```
namespace: kabanero
```

Convert the Role and RoleBinding to ClusterRole and ClusterRoleBinding

## Create All In One Yaml

```
cat knative-eventing.yaml > operators.yaml; echo "---" >> operators.yaml
cat knative-serving.yaml >> operators.yaml; echo "---" >> operators.yaml
cat tekton-operator.yaml >> operators.yaml; echo "---" >> operators.yaml
cat kabanero-operator.yaml >> operators.yaml; echo "---" >> operators.yaml
```

## Update Release Files

Set the release in config/samples/full.yaml and config/samples/simple.yaml:
```
  version: <version identifier>
```

## Create 'latest' link

```
# Remove existing
rm deploy/release/latest
# Create symlink
cd deploy/release
ln -s deploy/releases/0.0.2/ latest
```
