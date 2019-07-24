IMAGE ?= kabanero/kabanero-operator:latest

.PHONY: build deploy

build: dependencies
	go install ./cmd/manager

build-image: dependencies
	operator-sdk build ${IMAGE}
	# This is a workaround until manfistival can interact with the virtual file system
	docker build -t ${IMAGE} .

push-image: build-image
	docker push ${IMAGE}

generate:
	operator-sdk generate k8s
	operator-sdk generate openapi

install:
	kubectl apply -f deploy/crds/
	
deploy: 
	kubectl create namespace kabanero || true
	sed -i '' -e 's!image: kabanero/kabanero-operator:latest!image: ${IMAGE}!' deploy/operator.yaml
	kubectl -n kabanero apply -f deploy/

dependencies:
	go get -u github.com/golang/dep/cmd/dep
	dep ensure

# Requires https://github.com/pmezard/licenses
dependency-report: dependencies
	go get -u github.com/pmezard/licenses
	licenses ./pkg/... | cut -c49- > 3RD_PARTY