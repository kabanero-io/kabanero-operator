# The Docker image in format repository:tag. Repository may contain a remote reference.
# Override in order to customize
IMAGE ?= kabanero-operator:latest

# Computed repository name (no tag) including repository host/path reference
REPOSITORY=$(firstword $(subst :, ,${IMAGE}))

.PHONY: build deploy build-image push-image

build:
	go install ./cmd/manager

build-image: generate
	operator-sdk build ${IMAGE}
	# This is a workaround until manfistival can interact with the virtual file system
	docker build -t ${IMAGE} --build-arg IMAGE=${IMAGE} .

push-image:
ifneq "$(IMAGE)" "kabanero-operator:latest"
	# Default push
	docker push $(IMAGE)

ifdef TRAVIS_TAG
	# This is a Travis tag build. Pushing using Docker tag TRAVIS_TAG
	docker tag $(IMAGE) $(REPOSITORY):$(TRAVIS_TAG)
	docker push $(REPOSITORY):$(TRAVIS_TAG)
endif

ifdef TRAVIS_BRANCH
	# This is a Travis branch build. Pushing using Docker tag TRAVIS_BRANCH
	docker tag $(IMAGE) $(REPOSITORY):$(TRAVIS_BRANCH)
	docker push $(REPOSITORY):$(TRAVIS_BRANCH)
endif
endif

generate:
	operator-sdk generate k8s
	operator-sdk generate openapi

install:
	kubectl config set-context $$(kubectl config current-context) --namespace=kabanero
	kubectl apply -f deploy/crds/kabanero_kabanero_crd.yaml
	kubectl apply -f deploy/crds/kabanero_collection_crd.yaml
	
deploy: 
	kubectl create namespace kabanero || true

ifneq "$(IMAGE)" "kabanero-operator:latest"
	# By default there is no image pull policy for local image. However, for other images
	# substitute current image name, and update the pull policy to always pull images.
	sed -i.bak -e 's!imagePullPolicy: Never!imagePullPolicy: Always!' deploy/operator.yaml
	sed -i.bak -e 's!image: kabanero-operator:latest!image: ${IMAGE}!' deploy/operator.yaml
endif
	rm deploy/operator.yaml.bak
	kubectl config set-context $$(kubectl config current-context) --namespace=kabanero
	kubectl apply -f deploy/

dependencies: 
ifeq (, $(shell which dep))
	go get -u github.com/golang/dep/cmd/dep
endif
	dep ensure

	# Remove some creative commons licensed tests/samples
	rm vendor/golang.org/x/net/http2/h2demo/tmpl.go
	rm -r vendor/golang.org/x/text/internal/testtext

# Requires https://github.com/pmezard/licenses
dependency-report: 
	go get -u github.com/pmezard/licenses
	licenses ./pkg/... | cut -c49- > 3RD_PARTY
