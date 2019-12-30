# kabanero-operator
The following README pertains to kabanero-operator development.  If you are trying to use the operator to try Kabanero on your cluster, please see the [install instructions](https://kabanero.io/docs/ref/general/installing-kabanero-foundation.html).

# Status
[![Build Status](https://travis-ci.org/kabanero-io/kabanero-operator.svg?branch=master)](https://travis-ci.org/kabanero-io/kabanero-operator)

The Kabanero operator is developed using `operator-sdk` version 0.11.0.

## Clone the Kabanero operator

```
git clone https://github.com/kabanero-io/kabanero-operator
cd kabanero-operator
```

# Quickstart - OpenShift 3.11 / OKD 3.11

We recommend you follow the install instructions referenced above to set up your cluster for the first time.  If you would rather set it up manually, please continue with the following steps:

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
[admin@openshift kabanero-operator]# kubectl get pods -n kabanero
NAME                                                        READY     STATUS    RESTARTS   AGE
controller-manager-0                                        1/1       Running   2          2d
kabanero-operator-7974456fc-jc2dl                           1/1       Running   0          27m
knative-eventing-operator-67cdf5dc9f-s4bth                  1/1       Running   0          2d
knative-serving-operator-b64558bbc-9ndb5                    1/1       Running   0          2d
openshift-pipelines-operator-66c4d787cf-f2xwt               1/1       Running   0          2d
```

## Deploy the sample

```
kubectl apply -n kabanero -f config/samples/default.yaml
```

kubectl get pods -n kabanero
```
(example command output)
[admin@openshift kabanero-operator]# kubectl get pods -n kabanero
NAME                                                        READY     STATUS    RESTARTS   AGE
appsody-operator-79ccd57895-j7d9s                           1/1       Running   0          2d
controller-manager-0                                        1/1       Running   2          2d
kabanero-cli-679cbddb4f-gt5b6                               1/1       Running   0          2d
kabanero-landing-775584bbd4-4m9zb                           1/1       Running   0          2d
kabanero-operator-7974456fc-jc2dl                           1/1       Running   0          27m
knative-eventing-operator-67cdf5dc9f-s4bth                  1/1       Running   0          2d
knative-serving-operator-b64558bbc-9ndb5                    1/1       Running   0          2d
openshift-pipelines-operator-66c4d787cf-f2xwt               1/1       Running   0          2d
```

# Quickstart - minikube

Kabanero is not currently supported on Minikube, due to the resource requirements of its dependencies (Istio, Knative and Tekton) and due to the Kabanero-operator's dependencies on OpenShift types like `Routes`.

Some folks have had success installing on Minishift.  For example, see https://github.com/nastacio/kabanero-minishift.
