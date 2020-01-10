# kabanero-operator
The following README pertains to kabanero-operator development.  If you are trying to use the operator to try Kabanero on your cluster, please see the [install instructions](https://kabanero.io/docs/ref/general/installation/installing-kabanero-foundation.html).

# Status
[![Build Status](https://travis-ci.org/kabanero-io/kabanero-operator.svg?branch=master)](https://travis-ci.org/kabanero-io/kabanero-operator)

The Kabanero operator is developed using `operator-sdk` version 0.11.0.

## Clone the Kabanero operator

```
git clone https://github.com/kabanero-io/kabanero-operator
cd kabanero-operator
```

# Quickstart - OpenShift Container Platform (OCP) 4.2

We recommend you follow the install instructions referenced above to set up your cluster for the first time.  If you would rather set it up manually, please continue with the following steps:

## Login
(example)

```
oc login -u admin -p admin https://openshift.my.com:8443/
```

## Deploy Prerequisite operators:

The following operators need to be installed at the cluster scope:
* [OpenShift Serverless Operator](https://docs.openshift.com/container-platform/4.2/serverless/installing-openshift-serverless.html)
* OpenShift Pipelines Operator (from community-operators)
* Appsody Operator (from certified-operators)

## Build images, create catalogsource and OLM subscription

You will need to examine the `Makefile` and set any necessary variables to push your container images to the correct repository.

```
make build-image
make push-image
make deploy-olm
```

## Check that the operator pods are in Running state

```
kubectl get pods -n kabanero
```
(example command output)

```
[admin@openshift kabanero-operator]# kubectl get pods -n kabanero
NAME                                                       READY   STATUS    RESTARTS   AGE
kabanero-operator-659d7f84bb-v9jsp                         1/1     Running   0          3h5m
```

## Deploy the sample

```
kubectl apply -n kabanero -f config/samples/default.yaml
```

kubectl get pods -n kabanero
```
(example command output)
[admin@openshift kabanero-operator]# kubectl get pods -n kabanero
NAME                                                       READY   STATUS    RESTARTS   AGE
kabanero-cli-58f96db965-j97gd                              1/1     Running   0          3h4m
kabanero-landing-84b99fbcbf-z8zvx                          1/1     Running   0          3h4m
kabanero-operator-659d7f84bb-v9jsp                         1/1     Running   0          3h5m
kabanero-operator-admission-webhook-775668455c-j4nkf       1/1     Running   1          3h4m
kabanero-operator-collection-controller-6757dbc9bc-4tsht   1/1     Running   0          33m
```

# Quickstart - minikube

Kabanero is not currently supported on Minikube, due to the resource requirements of its dependencies (Istio, Knative and Tekton) and due to the Kabanero-operator's dependencies on OpenShift types like `Routes`.

Some folks have had success installing on Minishift.  For example, see https://github.com/nastacio/kabanero-minishift.

# Quickstart - OpenShift 3.11 / OKD 3.11

Please use the `release-0.2` branch when using OpenShift or OKD v3.11.
https://github.com/kabanero-io/kabanero-operator/tree/release-0.2