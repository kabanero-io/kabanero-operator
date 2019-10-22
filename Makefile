# The Docker image in format repository:tag. Repository may contain a remote reference.
# Override in order to customize
IMAGE ?= kabanero-operator:latest
REGISTRY_IMAGE ?= kabanero-operator-registry:latest

# Computed repository name (no tag) including repository host/path reference
REPOSITORY=$(firstword $(subst :, ,${IMAGE}))
REGISTRY_REPOSITORY=$(firstword $(subst :, ,${REGISTRY_IMAGE}))

.PHONY: build deploy deploy-olm build-image push-image

build: generate
	go install ./cmd/manager

build-image: generate
  # These commands were taken from operator-sdk 0.8.1.  The sdk did not let us
  # pass the ldflags option.  The advice from operator-sdk was to run the 
  # commands separately here.
  # operator-sdk build ${IMAGE}
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o build/_output/bin/kabanero-operator -gcflags "all=-trimpath=$(GOPATH)" -asmflags "all=-trimpath=$(GOPATH)" -ldflags "-X main.GitTag=$(TRAVIS_TAG) -X main.GitCommit=$(TRAVIS_COMMIT) -X main.GitRepoSlug=$(TRAVIS_REPO_SLUG) -X main.BuildDate=`date -u +%Y%m%d.%H%M%S`" github.com/kabanero-io/kabanero-operator/cmd/manager
	docker build -f build/Dockerfile -t ${IMAGE} .
  # This is a workaround until manfistival can interact with the virtual file system
	docker build -t ${IMAGE} --build-arg IMAGE=${IMAGE} .
  # Build an OLM private registry for Kabanero
	mkdir -p build/registry
	cp -R registry/manifests build/registry/
	cp registry/Dockerfile build/registry/Dockerfile
	cp deploy/crds/kabanero_v1alpha1_*_crd.yaml build/registry/manifests/kabanero-operator/0.3.0/
	sed -e "s!kabanero/kabanero-operator:latest!${IMAGE}!" registry/manifests/kabanero-operator/0.3.0/kabanero-operator.v0.3.0.clusterserviceversion.yaml > build/registry/manifests/kabanero-operator/0.3.0/kabanero-operator.v0.3.0.clusterserviceversion.yaml
	docker build -t ${REGISTRY_IMAGE} -f build/registry/Dockerfile build/registry/
	rm -R build/registry

push-image:
ifneq "$(IMAGE)" "kabanero-operator:latest"
  # Default push
	docker push $(IMAGE)
	docker push $(REGISTRY_IMAGE)

ifdef TRAVIS_TAG
  # This is a Travis tag build. Pushing using Docker tag TRAVIS_TAG
	docker tag $(IMAGE) $(REPOSITORY):$(TRAVIS_TAG)
	docker push $(REPOSITORY):$(TRAVIS_TAG)
	docker tag $(REGISTRY_IMAGE) $(REGISTRY_REPOSITORY):$(TRAVIS_TAG)
	docker push $(REGISTRY_REPOSITORY):$(TRAVIS_TAG)
endif

ifdef TRAVIS_BRANCH
  # This is a Travis branch build. Pushing using Docker tag TRAVIS_BRANCH
	docker tag $(IMAGE) $(REPOSITORY):$(TRAVIS_BRANCH)
	docker push $(REPOSITORY):$(TRAVIS_BRANCH)
	docker tag $(REGISTRY_IMAGE) $(REGISTRY_REPOSITORY):$(TRAVIS_BRANCH)
	docker push $(REGISTRY_REPOSITORY):$(TRAVIS_BRANCH)
endif
endif

test: 
	go test ./cmd/... ./pkg/... 

format:
	go fmt ./cmd/... ./pkg/...

generate:
	operator-sdk generate k8s
	operator-sdk generate openapi
	go generate ./pkg/assets

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
	rm deploy/operator.yaml.bak || true
	kubectl config set-context $$(kubectl config current-context) --namespace=kabanero
	kubectl apply -f deploy/

deploy-olm:
	kubectl create namespace kabanero || true
	sed -i.bak -e "s!image: KABANERO_REGISTRY_IMAGE!image: ${REGISTRY_IMAGE}!" deploy/olm/01-catalog-source.yaml
	rm deploy/olm/01-catalog-source.yaml.bak || true
	kubectl apply -f deploy/olm/

check: format build #test

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
