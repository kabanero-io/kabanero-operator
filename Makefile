.PHONY: build deploy

build:
	go install ./cmd/manager

build-image: 
	operator-sdk build kabanero-operator:latest

generate:
	operator-sdk generate k8s

install:
	kubectl apply -f deploy/crds/kabanero_v1alpha1_kabanero_crd.yaml
	
deploy: 
	kubectl create namespace kabanero || true
	kubectl -n kabanero apply -f deploy/dependencies.yaml
