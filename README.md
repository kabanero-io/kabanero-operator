# kabanero-operator
The following README pertains to kabanero-operator development.  If you are trying to use the operator to try Kabanero on your cluster, please see the [install instructions](https://kabanero.io/docs/ref/general/installing-kabanero-foundation.html).

# Status
[![Build Status](https://travis-ci.org/kabanero-io/kabanero-operator.svg?branch=master)](https://travis-ci.org/kabanero-io/kabanero-operator)

The Kabanero operator is developed using `operator-sdk` version 0.8.1.

## Clone the Kabanero operator

```
git clone https://github.com/kabanero-io/kabanero-operator
cd kabanero-operator
```

# Quickstart - OpenShift 3.11 / OKD 3.11

We recomment you follow the install instructions reference above to set up your cluster the first time.  If you would rather set up manually, please continue with the following steps:

## Login
(example)

```
oc login -u admin -p admin https://openshift.my.com:8443/
```

## Deploy Istio

Kabanero on OKD 3.11 / OpenShift 3.11 has been tested with Istio version 1.1.7.  Follow the instructions here to deploy the [Quick Start Evaluation Install](https://archive.istio.io/v1.1/docs/setup/kubernetes/install/kubernetes/).  We suggest you pick the Permissive Mutual TLS configuration to get started.

## Create Kabanero CRDs

```
make install
```

## Deploy the CRDs and some of the other controllers (Knative, Tekton)

```
make deploy
```

## Check that the operator pods are in Running state

```
kubectl get pods -n kabanero
```
(example command output)

```
[root@openshist kabanero-operator]# kubectl get pods -n kabanero
NAME                                           READY     STATUS    RESTARTS   AGE
kabanero-operator-dd997974-5nfj4               1/1       Running   0          5m
knative-eventing-operator-658765d7d6-pq5bk     1/1       Running   0          5m
knative-serving-operator-8c7858985-plvh8       1/1       Running   0          5m
openshift-pipelines-operator-c56876c69-hf74v   1/1       Running   0          5m
```

## Deploy the sample

```
kubectl apply -n kabanero -f config/samples/default.yaml
```

kubectl get pods -n kabanero
```
(example command output)
[root@openshist kabanero-operator]# kubectl get pods -n kabanero
NAME                                           READY     STATUS    RESTARTS   AGE
kabanero-operator-dd997974-4cqnt               1/1       Running   0          2m
knative-eventing-operator-658765d7d6-6ptqc     1/1       Running   0          2m
knative-serving-operator-8c7858985-v7zrd       1/1       Running   0          2m
openshift-pipelines-operator-c56876c69-6jwqw   1/1       Running   0          2m
tekton-pipelines-controller-5576fbb979-mv8hp   1/1       Running   0          55s
tekton-pipelines-webhook-78bf9c5f46-9pzlj      1/1       Running   0          55s
```

# Quickstart - minikube

Kabanero is not currently supported on Minikube, due to the resource requirements of its dependencies (Istio, Knative and Tekton) and due to the Kabanero-operator's dependencies on OpenShift types like `Routes`.

Some folks have had success installing on Minishift.  For example, see https://github.com/nastacio/kabanero-minishift.
