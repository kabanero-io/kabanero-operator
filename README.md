# kabanero-operator
The Kabanero platform operator

# Status
[![Build Status](https://travis-ci.org/kabanero-io/kabanero-operator.svg?branch=master)](https://travis-ci.org/kabanero-io/kabanero-operator)

# Quickstart - minikube

Create a minikube instance: 
```
minikube start --memory=8192 --cpus=4 \
  --kubernetes-version=v1.12.0 \
  --vm-driver=hyperkit \
  --disk-size=30g \
  --extra-config=apiserver.enable-admission-plugins="LimitRanger,NamespaceExists,NamespaceLifecycle,ResourceQuota,ServiceAccount,DefaultStorageClass,MutatingAdmissionWebhook"

kubectl apply --filename https://github.com/knative/serving/releases/download/v0.4.0/istio-crds.yaml
curl -L https://github.com/knative/serving/releases/download/v0.4.0/istio.yaml | sed 's/LoadBalancer/NodePort/' | kubectl apply --filename -
```

## Deploy the Operators

```
kubectl apply -f https://raw.githubusercontent.com/kabanero-io/kabanero-operator/master/deploy/releases/latest/kabanero-operators.yaml
```

## Deploy the sample

```
kubectl apply -n kabanero -f https://raw.githubusercontent.com/kabanero-io/kabanero-operator/master/config/samples/full.yaml
```

# Quickstart - OpenShift

## Login
(example)

```
oc login -u admin -p admin https://openshift.my.com:8443/
```

## Deploy Istio

```
kubectl apply --filename https://github.com/knative/serving/releases/download/v0.4.0/istio-crds.yaml
curl -L https://github.com/knative/serving/releases/download/v0.4.0/istio.yaml | sed 's/LoadBalancer/NodePort/' | kubectl apply --filename -
```

## Deploy the Operators

```
kubectl apply -f https://raw.githubusercontent.com/kabanero-io/kabanero-operator/master/deploy/releases/latest/kabanero-operators.yaml
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
kubectl apply -n kabanero -f https://raw.githubusercontent.com/kabanero-io/kabanero-operator/master/config/samples/full.yaml
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
