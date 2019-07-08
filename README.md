# kabanero-operator
The Kabanero platform operator

# Status
[![Build Status](https://travis.com/kabanero-io/kabanero-operator.svg?token=JCs1u1Thd9q5ND5Yz3TK&branch=master)](https://travis.com/kabanero-io/kabanero-operator)

## Quickstart
Create a minikube instance: 
```
minikube start --memory=8192 --cpus=4 \
  --kubernetes-version=v1.12.0 \
  --vm-driver=hyperkit \
  --disk-size=30g \
  --extra-config=apiserver.enable-admission-plugins="LimitRanger,NamespaceExists,NamespaceLifecycle,ResourceQuota,ServiceAccount,DefaultStorageClass,MutatingAdmissionWebhook"

kubectl apply --filename https://github.com/knative/serving/releases/download/v0.4.0/istio-crds.yaml &&
curl -L https://github.com/knative/serving/releases/download/v0.4.0/istio.yaml \
  | sed 's/LoadBalancer/NodePort/' \
  | kubectl apply --filename -
```

### Deploy the Operator and Dependencies
```
# Create Kabanero CRDs
make install

# Deploy the CRDs and some of the other controllers
make deploy
```


### Deploy the sample
```
kubectl apply -n kabanero -f config/samples/full.yaml
```
