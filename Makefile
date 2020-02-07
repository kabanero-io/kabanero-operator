# The Docker image in format repository:tag. Repository may contain a remote reference.
# Override in order to customize
IMAGE ?= kabanero-operator:latest
REGISTRY_IMAGE ?= kabanero-operator-registry:latest
WEBHOOK_IMAGE ?= kabanero-operator-admission-webhook:latest
COLLECTION_CTRLR_IMAGE ?= kabanero-operator-collection-controller:latest
STACK_CTRLR_IMAGE ?= kabanero-operator-stack-controller:latest

# For integration testing
# INTERNAL_REGISTRY: the public facing registry url. Set TRUE to enable and find the default address, or manually set to address itself
# INTERNAL_REGISTRY_SVC: the internal service image pull address. If not set, set to the default
INTERNAL_REGISTRY ?=
INTERNAL_REGISTRY_SVC ?=
ifdef INTERNAL_REGISTRY
ifeq ($(INTERNAL_REGISTRY),TRUE)
INTERNAL_REGISTRY := $(shell kubectl get route default-route -n openshift-image-registry --template='{{ .spec.host }}')
ifndef INTERNAL_REGISTRY_SVC
INTERNAL_REGISTRY_SVC=image-registry.openshift-image-registry.svc:5000
endif
# Public registry references
IMAGE=${INTERNAL_REGISTRY}/kabanero/kabanero-operator:latest
REGISTRY_IMAGE=${INTERNAL_REGISTRY}/openshift-marketplace/kabanero-operator-registry:latest
WEBHOOK_IMAGE=${INTERNAL_REGISTRY}/kabanero/kabanero-operator-admission-webhook:latest
COLLECTION_CTRLR_IMAGE=${INTERNAL_REGISTRY}/kabanero/kabanero-operator-collection-controller:latest
STACK_CTRLR_IMAGE=${INTERNAL_REGISTRY}/kabanero/kabanero-operator-stack-controller:latest
# Internal service registry references
IMAGE_SVC=${INTERNAL_REGISTRY_SVC}/kabanero/kabanero-operator:latest
REGISTRY_IMAGE_SVC=${INTERNAL_REGISTRY_SVC}/openshift-marketplace/kabanero-operator-registry:latest
WEBHOOK_IMAGE_SVC=${INTERNAL_REGISTRY_SVC}/kabanero/kabanero-operator-admission-webhook:latest
COLLECTION_CTRLR_IMAGE_SVC=${INTERNAL_REGISTRY_SVC}/kabanero/kabanero-operator-collection-controller:latest
STACK_CTRLR_IMAGE_SVC=${INTERNAL_REGISTRY_SVC}/kabanero/kabanero-operator-stack-controller:latest
endif
endif


# Computed repository name (no tag) including repository host/path reference
REPOSITORY=$(firstword $(subst :, ,${IMAGE}))
REGISTRY_REPOSITORY=$(firstword $(subst :, ,${REGISTRY_IMAGE}))
WEBHOOK_REPOSITORY=$(firstword $(subst :, ,${WEBHOOK_IMAGE}))
COLLECTION_CTRLR_REPOSITORY=$(firstword $(subst :, ,${COLLECTION_CTRLR_IMAGE}))
STACK_CTRLR_REPOSITORY=$(firstword $(subst :, ,${STACK_CTRLR_IMAGE}))


# Current release (used for CSV management)
CURRENT_RELEASE=0.6.0

# OS detection
ifeq ($(OS),Windows_NT)
	detected_OS := windows
else
	detected_OS := $(shell uname)
ifeq ($(detected_OS),Darwin)
	detected_OS := macos
endif
endif

.PHONY: build deploy deploy-olm build-image push-image int-test-install int-test-collections int-test-uninstall int-test-lifecycle

build: generate
	GO111MODULE=on go install ./cmd/manager
	GO111MODULE=on go install ./cmd/manager/collection
	GO111MODULE=on go install ./cmd/manager/stack
	GO111MODULE=on go install ./cmd/admission-webhook

build-image: generate
  # These commands were taken from operator-sdk 0.8.1.  The sdk did not let us
  # pass the ldflags option.  The advice from operator-sdk was to run the 
  # commands separately here.
  # operator-sdk build ${IMAGE}
	GO111MODULE=on CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o build/_output/bin/kabanero-operator -gcflags "all=-trimpath=$(GOPATH)" -asmflags "all=-trimpath=$(GOPATH)" -ldflags "-X main.GitTag=$(TRAVIS_TAG) -X main.GitCommit=$(TRAVIS_COMMIT) -X main.GitRepoSlug=$(TRAVIS_REPO_SLUG) -X main.BuildDate=`date -u +%Y%m%d.%H%M%S`" github.com/kabanero-io/kabanero-operator/cmd/manager
	GO111MODULE=on CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o build/_output/bin/kabanero-operator-collection-controller -gcflags "all=-trimpath=$(GOPATH)" -asmflags "all=-trimpath=$(GOPATH)" -ldflags "-X main.GitTag=$(TRAVIS_TAG) -X main.GitCommit=$(TRAVIS_COMMIT) -X main.GitRepoSlug=$(TRAVIS_REPO_SLUG) -X main.BuildDate=`date -u +%Y%m%d.%H%M%S`" github.com/kabanero-io/kabanero-operator/cmd/manager/collection
	GO111MODULE=on CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o build/_output/bin/kabanero-operator-stack-controller -gcflags "all=-trimpath=$(GOPATH)" -asmflags "all=-trimpath=$(GOPATH)" -ldflags "-X main.GitTag=$(TRAVIS_TAG) -X main.GitCommit=$(TRAVIS_COMMIT) -X main.GitRepoSlug=$(TRAVIS_REPO_SLUG) -X main.BuildDate=`date -u +%Y%m%d.%H%M%S`" github.com/kabanero-io/kabanero-operator/cmd/manager/stack
	GO111MODULE=on CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o build/_output/bin/admission-webhook -gcflags "all=-trimpath=$(GOPATH)" -asmflags "all=-trimpath=$(GOPATH)" -ldflags "-X main.GitTag=$(TRAVIS_TAG) -X main.GitCommit=$(TRAVIS_COMMIT) -X main.GitRepoSlug=$(TRAVIS_REPO_SLUG) -X main.BuildDate=`date -u +%Y%m%d.%H%M%S`" github.com/kabanero-io/kabanero-operator/cmd/admission-webhook

	docker build -f build/Dockerfile -t ${IMAGE} .

  # Build the Kananero collection controller image.
	docker build -f build/Dockerfile-collection-controller -t ${COLLECTION_CTRLR_IMAGE} .

	# Build the Kananero stack controller image.
	docker build -f build/Dockerfile-stack-controller -t ${STACK_CTRLR_IMAGE} .

  # Build the admission webhook.
	docker build -f build/Dockerfile-webhook -t ${WEBHOOK_IMAGE} .

  # Build an OLM private registry for Kabanero
	mkdir -p build/registry
	cp LICENSE build/registry/LICENSE
	cp -R registry/manifests build/registry/
	cp registry/Dockerfile build/registry/Dockerfile
	cp deploy/crds/kabanero.io_kabaneros_crd.yaml deploy/crds/kabanero.io_collections_crd.yaml deploy/crds/kabanero.io_stacks_crd.yaml build/registry/manifests/kabanero-operator/$(CURRENT_RELEASE)/

ifdef INTERNAL_REGISTRY
	sed -e "s!kabanero/kabanero-operator:.*!${IMAGE_SVC}!" registry/manifests/kabanero-operator/$(CURRENT_RELEASE)/kabanero-operator.v$(CURRENT_RELEASE).clusterserviceversion.yaml > build/registry/manifests/kabanero-operator/$(CURRENT_RELEASE)/kabanero-operator.v$(CURRENT_RELEASE).clusterserviceversion.yaml
else
	sed -e "s!kabanero/kabanero-operator:.*!${IMAGE}!" registry/manifests/kabanero-operator/$(CURRENT_RELEASE)/kabanero-operator.v$(CURRENT_RELEASE).clusterserviceversion.yaml > build/registry/manifests/kabanero-operator/$(CURRENT_RELEASE)/kabanero-operator.v$(CURRENT_RELEASE).clusterserviceversion.yaml
endif

	docker build -t ${REGISTRY_IMAGE} -f build/registry/Dockerfile build/registry/

  # If we're doing a Travis build, need to build a second image because the CSV
  # in the registry image has to point to the tagged operator image.
ifdef TRAVIS_TAG
	sed -e "s!kabanero/kabanero-operator:.*!${REPOSITORY}:${TRAVIS_TAG}!" registry/manifests/kabanero-operator/$(CURRENT_RELEASE)/kabanero-operator.v$(CURRENT_RELEASE).clusterserviceversion.yaml > build/registry/manifests/kabanero-operator/$(CURRENT_RELEASE)/kabanero-operator.v$(CURRENT_RELEASE).clusterserviceversion.yaml
	docker build -t ${REGISTRY_REPOSITORY}:${TRAVIS_TAG} -f build/registry/Dockerfile build/registry/
endif

ifdef TRAVIS_BRANCH
	sed -e "s!kabanero/kabanero-operator:.*!${REPOSITORY}:${TRAVIS_BRANCH}!" registry/manifests/kabanero-operator/$(CURRENT_RELEASE)/kabanero-operator.v$(CURRENT_RELEASE).clusterserviceversion.yaml > build/registry/manifests/kabanero-operator/$(CURRENT_RELEASE)/kabanero-operator.v$(CURRENT_RELEASE).clusterserviceversion.yaml
	docker build -t ${REGISTRY_REPOSITORY}:${TRAVIS_BRANCH} -f build/registry/Dockerfile build/registry/
endif

	rm -R build/registry

push-image:
ifneq "$(IMAGE)" "kabanero-operator:latest"
  # Default push.  Make sure the namespace is there in case using local registry
	kubectl create namespace kabanero || true
	docker push $(IMAGE)
	docker push $(COLLECTION_CTRLR_IMAGE)
	docker push $(STACK_CTRLR_IMAGE)
	docker push $(REGISTRY_IMAGE)
	docker push $(WEBHOOK_IMAGE)

ifdef TRAVIS_TAG
  # This is a Travis tag build. Pushing using Docker tag TRAVIS_TAG
	docker tag $(IMAGE) $(REPOSITORY):$(TRAVIS_TAG)
	docker push $(REPOSITORY):$(TRAVIS_TAG)
	docker tag $(WEBHOOK_IMAGE) $(WEBHOOK_REPOSITORY):$(TRAVIS_TAG)
	docker push $(WEBHOOK_REPOSITORY):$(TRAVIS_TAG)
	docker push $(REGISTRY_REPOSITORY):$(TRAVIS_TAG)
	docker tag $(COLLECTION_CTRLR_IMAGE) $(COLLECTION_CTRLR_REPOSITORY):$(TRAVIS_TAG)
	docker push $(COLLECTION_CTRLR_REPOSITORY):$(TRAVIS_TAG)
	docker tag $(STACK_CTRLR_IMAGE) $(STACK_CTRLR_REPOSITORY):$(TRAVIS_TAG)
	docker push $(STACK_CTRLR_REPOSITORY):$(TRAVIS_TAG)
endif

ifdef TRAVIS_BRANCH
  # This is a Travis branch build. Pushing using Docker tag TRAVIS_BRANCH
	docker tag $(IMAGE) $(REPOSITORY):$(TRAVIS_BRANCH)
	docker push $(REPOSITORY):$(TRAVIS_BRANCH)
	docker tag $(WEBHOOK_IMAGE) $(WEBHOOK_REPOSITORY):$(TRAVIS_BRANCH)
	docker push $(WEBHOOK_REPOSITORY):$(TRAVIS_BRANCH)
	docker push $(REGISTRY_REPOSITORY):$(TRAVIS_BRANCH)
	docker tag $(COLLECTION_CTRLR_IMAGE) $(COLLECTION_CTRLR_REPOSITORY):$(TRAVIS_BRANCH)
	docker push $(COLLECTION_CTRLR_REPOSITORY):$(TRAVIS_BRANCH)
	docker tag $(STACK_CTRLR_IMAGE) $(STACK_CTRLR_REPOSITORY):$(TRAVIS_BRANCH)
	docker push $(STACK_CTRLR_REPOSITORY):$(TRAVIS_BRANCH)
endif
endif

test: 
	GO111MODULE=on go test ./cmd/... ./pkg/... 

format:
	GO111MODULE=on go fmt ./cmd/... ./pkg/...

generate:
	GO111MODULE=on operator-sdk generate k8s
	GO111MODULE=on operator-sdk generate openapi
	GO111MODULE=on go generate ./pkg/assets

install:
	kubectl config set-context $$(kubectl config current-context) --namespace=kabanero
	kubectl apply -f deploy/crds/kabanero.io_kabaneros_crd.yaml
	kubectl apply -f deploy/crds/kabanero.io_collections_crd.yaml
	kubectl apply -f deploy/crds/kabanero.io_stacks_crd.yaml

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

	kubectl apply -f deploy/olm/

# Update deployment to correct image 
ifdef INTERNAL_REGISTRY
	sed -e "s!image: KABANERO_REGISTRY_IMAGE!image: ${REGISTRY_IMAGE_SVC}!" deploy/olm/01-catalog-source.yaml | kubectl apply -f -
else
	sed -e "s!image: KABANERO_REGISTRY_IMAGE!image: ${REGISTRY_IMAGE}!" deploy/olm/01-catalog-source.yaml | kubectl apply -f -
endif

check: format build test

dependencies: 
ifeq (, $(shell which dep))
	GO111MODULE=on go get -u github.com/golang/dep/cmd/dep
endif
	dep ensure

  # Remove some creative commons licensed tests/samples
	rm vendor/golang.org/x/net/http2/h2demo/tmpl.go
	rm -r vendor/golang.org/x/text/internal/testtext

# Requires https://github.com/mitchellh/golicense
# Note that the tool currently fails when a license is not found.  Since
# this is currently the case for several dependencies, the file must be
# inspected manually, and we append || true to the command.
dependency-report:
ifndef GITHUB_TOKEN
	$(error GITHUB_TOKEN must be set to a PAT to run the license check)
endif
	mkdir -p build/bin
	curl -L https://github.com/mitchellh/golicense/releases/download/v0.2.0/golicense_0.2.0_$(detected_OS)_x86_64.tar.gz | tar -C build/bin -xzf - golicense
	build/bin/golicense -plain ./license-rules.json build/_output/bin/admission-webhook build/_output/bin/kabanero-operator build/_output/bin/kabanero-operator-collection-controller build/_output/bin/kabanero-operator-stack-controller > 3RD_PARTY || true
	rm build/bin/golicense

# Integration Tests
# Requires kube login context an existing cluster
# Requires internal registry with default route
# Requires jq

# Install Test
int-test-install: creds build-image push-image int-install int-config

creds:
	tests/00-credentials.sh

int-install:
# Update deployment to correct image 
ifdef INTERNAL_REGISTRY
	sed -e "s!image: kabanero/kabanero-operator-registry:.*!image: ${REGISTRY_IMAGE_SVC}!" deploy/kabanero-subscriptions.yaml > /tmp/kabanero-subscriptions.yaml
else
	sed -e "s!image: kabanero/kabanero-operator-registry:.*!image: ${REGISTRY_IMAGE}!" deploy/kabanero-subscriptions.yaml > /tmp/kabanero-subscriptions.yaml
endif

	KABANERO_SUBSCRIPTIONS_YAML=/tmp/kabanero-subscriptions.yaml KABANERO_CUSTOMRESOURCES_YAML=deploy/kabanero-customresources.yaml deploy/install.sh

int-config:
# Update config to correct image
# Update config to correct image
ifdef INTERNAL_REGISTRY
	sed -e "s!image: kabanero/kabanero-operator-admission-webhook:.*!image: ${WEBHOOK_IMAGE_SVC}!; s!image: kabanero/kabanero-operator-collection-controller:.*!image: ${COLLECTION_CTRLR_IMAGE_SVC}!; s!image: kabanero/kabanero-operator-stack-controller:.*!image: ${STACK_CTRLR_IMAGE_SVC}!" config/samples/full.yaml | kubectl -n kabanero apply -f -
else
	sed -e "s!image: kabanero/kabanero-operator-admission-webhook:.*!image: ${WEBHOOK_IMAGE}!; s!image: kabanero/kabanero-operator-collection-controller:.*!image: ${COLLECTION_CTRLR_IMAGE}!; s!image: kabanero/kabanero-operator-stack-controller:.*!image: ${STACK_CTRLR_IMAGE}!" config/samples/full.yaml | kubectl -n kabanero apply -f -
endif

# Uninstall Test
int-test-uninstall: creds int-uninstall

int-uninstall:
	KABANERO_SUBSCRIPTIONS_YAML=deploy/kabanero-subscriptions.yaml KABANERO_CUSTOMRESOURCES_YAML=deploy/kabanero-customresources.yaml deploy/uninstall.sh

# Collections: Can be run in parallel ( -j ). Test manual pipeline run of collections.
int-test-collections: int-test-java-microprofile int-test-java-spring-boot2 int-test-nodejs int-test-nodejs-express int-test-nodejs-loopback

int-test-java-microprofile:
	tests/10-java-microprofile.sh

int-test-java-spring-boot2:
	tests/11-java-spring-boot2.sh

int-test-nodejs:
	tests/12-nodejs.sh

int-test-nodejs-express:
	tests/13-nodejs-express.sh

int-test-nodejs-loopback:
	tests/14-nodejs-loopback.sh

# Lifecycle: Lifecycle of Kabanero, Collection and owned objects
int-test-lifecycle: int-test-delete-pipeline int-test-delete-collection int-test-update-index int-test-deactivate-collection
int-test-delete-pipeline:
	tests/20-delete-pipeline.sh
int-test-delete-collection:
	tests/21-delete-collection.sh
int-test-update-index:
	tests/22-update-index.sh
int-test-deactivate-collection:
	tests/23-deactivate-collection.sh
