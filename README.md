# kabanero-operator
The Kabanero platform operator

# Status
[![Build Status](https://travis-ci.org/kabanero-io/kabanero-operator.svg?branch=master)](https://travis-ci.org/kabanero-io/kabanero-operator)

# Prerequisites

* [Go installed](https://golang.org/doc/install)

## Clone the Kabanero operator

(assume $GOHOME == $HOME)
```
mkdir -p $HOME/go/src/github.com/kabanero-io
cd $HOME/go/github.com/src/github.com/kabanero-io/
git clone https://github.com/kabanero-io/kabanero-operator
cd kabanero-operator
```

# Quickstart - minikube

Create a minikube instance: 
```
minikube start --memory=8192 --cpus=4 \
  --kubernetes-version=v1.12.0 \
  --vm-driver=hyperkit \
  --disk-size=30g \
  --extra-config=apiserver.enable-admission-plugins="LimitRanger,NamespaceExists,NamespaceLifecycle,ResourceQuota,ServiceAccount,DefaultStorageClass,MutatingAdmissionWebhook"

```
kubectl apply --filename https://github.com/knative/serving/releases/download/v0.4.0/istio-crds.yaml
curl -L https://github.com/knative/serving/releases/download/v0.4.0/istio.yaml | sed 's/LoadBalancer/NodePort/' | kubectl apply --filename -
```

## Create Kabanero CRDs

```
make install
```

## Deploy the CRDs and some of the other controllers

```
make deploy
```
## Deploy the sample

```
kubectl apply -n kabanero -f config/samples/full.yaml
```

# Quickstart - OpenShift

## Login
(example)

```
oc login -u admin -p admin https://openshift.my.com:8443/
```

## Create permissions for Istio

```
oc adm policy add-scc-to-user anyuid -z istio-ingress-service-account -n istio-system
oc adm policy add-scc-to-user anyuid -z default -n istio-system
oc adm policy add-scc-to-user anyuid -z prometheus -n istio-system
oc adm policy add-scc-to-user anyuid -z istio-egressgateway-service-account -n istio-system
oc adm policy add-scc-to-user anyuid -z istio-citadel-service-account -n istio-system
oc adm policy add-scc-to-user anyuid -z istio-ingressgateway-service-account -n istio-system
oc adm policy add-scc-to-user anyuid -z istio-cleanup-old-ca-service-account -n istio-system
oc adm policy add-scc-to-user anyuid -z istio-mixer-post-install-account -n istio-system
oc adm policy add-scc-to-user anyuid -z istio-mixer-service-account -n istio-system
oc adm policy add-scc-to-user anyuid -z istio-pilot-service-account -n istio-system
oc adm policy add-scc-to-user anyuid -z istio-sidecar-injector-service-account -n istio-system
oc adm policy add-cluster-role-to-user cluster-admin -z istio-galley-service-account -n istio-system
oc adm policy add-scc-to-user anyuid -z cluster-local-gateway-service-account -n istio-system
```

## Create permissions for the knative operator

```
oc adm policy add-cluster-role-to-user cluster-admin -z knative-eventing-operator -n kabanero
oc adm policy add-cluster-role-to-user cluster-admin -z knative-serving-operator -n kabanero
```

## Deploy Istio

```
kubectl apply --filename https://github.com/knative/serving/releases/download/v0.4.0/istio-crds.yaml
curl -L https://github.com/knative/serving/releases/download/v0.4.0/istio.yaml | sed 's/LoadBalancer/NodePort/' | kubectl apply --filename -
```

## Create Kabanero CRDs

```
make install
```

## Deploy the CRDs and some of the other controllers

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
kubectl apply -n kabanero -f config/samples/full.yaml
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
