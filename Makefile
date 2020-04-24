# The Docker image in format repository:tag. Repository may contain a remote reference.
# Override in order to customize:
#   ARCH would be the target architecture (amd64, ppc64le, s390x)
#   DOCKER_ID would be your docker user name
#   DOCKER_TAG would be the tag you want to use in the repository
ARCH ?= amd64


### TRAVIS_TAG build
ifdef TRAVIS_TAG
IMAGE = kabanero/kabanero-operator:$(TRAVIS_TAG)
REGISTRY_IMAGE = kabanero/kabanero-operator-registry:$(TRAVIS_TAG)
### TRAVIS_BRANCH build
else ifdef TRAVIS_BRANCH
IMAGE = kabanero/kabanero-operator:$(TRAVIS_BRANCH)
REGISTRY_IMAGE = kabanero/kabanero-operator-registry:$(TRAVIS_BRANCH)
### Personal Registry Build
else ifdef DOCKER_ID
DOCKER_TAG ?= latest
IMAGE = $(DOCKER_ID)/kabanero-operator:$(DOCKER_TAG)
REGISTRY_IMAGE = $(DOCKER_ID)/kabanero-operator-registry:$(DOCKER_TAG)
### Local cluster integration testing
else ifdef INTERNAL_REGISTRY
ifeq ($(INTERNAL_REGISTRY),TRUE)
INTERNAL_REGISTRY := $(shell kubectl get route default-route -n openshift-image-registry --template='{{ .spec.host }}')
endif
INTERNAL_REGISTRY_SVC ?= image-registry.openshift-image-registry.svc:5000
# Public facing registry references
IMAGE=$(INTERNAL_REGISTRY)/kabanero/kabanero-operator:latest
REGISTRY_IMAGE=$(INTERNAL_REGISTRY)/openshift-marketplace/kabanero-operator-registry:latest
# Internal facing service registry references
IMAGE_SVC=$(INTERNAL_REGISTRY_SVC)/kabanero/kabanero-operator:latest
REGISTRY_IMAGE_SVC=$(INTERNAL_REGISTRY_SVC)/openshift-marketplace/kabanero-operator-registry:latest
### Default
else
IMAGE ?= kabanero-operator:latest
REGISTRY_IMAGE ?= kabanero-operator-registry:latest
endif


# Get IMAGE with digest
IMAGE_REPO_DIGEST = $(shell docker image inspect $(IMAGE) --format="{{index .RepoDigests 0}}")
REGISTRY_IMAGE_REPO_DIGEST = $(shell docker image inspect $(REGISTRY_IMAGE) --format="{{index .RepoDigests 0}}")

# Computed repository name (no tag) including repository host/path reference
# Used to populate CSV for INTERNAL_REGISTRY case
REPOSITORY_SVC=$(shell echo $(IMAGE_SVC) | sed -r 's/:[0-9A-Za-z][0-9A-Za-z.-]{0,127}$$//g')
REPOSITORY_REGISTRY_SVC=$(shell echo $(REGISTRY_IMAGE_SVC) | sed -r 's/:[0-9A-Za-z][0-9A-Za-z.-]{0,127}$$//g')

# Get the SHA substring
# Used to populate CSV for INTERNAL_REGISTRY case
IMAGE_SHA = $(lastword $(subst @, ,$(IMAGE_REPO_DIGEST)))
REGISTRY_IMAGE_SHA = $(lastword $(subst @, ,$(REGISTRY_IMAGE_REPO_DIGEST)))

# Current release (used for CSV management)
CURRENT_RELEASE=0.9.0

# OS detection
ifeq ($(OS),Windows_NT)
	detected_OS := windows
else
	detected_OS := $(shell uname)
ifeq ($(detected_OS),Darwin)
	detected_OS := macos
endif
endif


.PHONY: build deploy deploy-olm build-image build-registry-image push-image push-registry-image push-manifest int-test-install int-test-collections int-test-uninstall int-test-lifecycle

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
	GO111MODULE=on CGO_ENABLED=0 GOOS=linux GOARCH=$(ARCH) go build -o build/_output/bin/kabanero-operator -gcflags "all=-trimpath=$(GOPATH)" -asmflags "all=-trimpath=$(GOPATH)" -ldflags "-X main.GitTag=$(TRAVIS_TAG) -X main.GitCommit=$(TRAVIS_COMMIT) -X main.GitRepoSlug=$(TRAVIS_REPO_SLUG) -X main.BuildDate=`date -u +%Y%m%d.%H%M%S`" github.com/kabanero-io/kabanero-operator/cmd/manager
	GO111MODULE=on CGO_ENABLED=0 GOOS=linux GOARCH=$(ARCH) go build -o build/_output/bin/kabanero-operator-collection-controller -gcflags "all=-trimpath=$(GOPATH)" -asmflags "all=-trimpath=$(GOPATH)" -ldflags "-X main.GitTag=$(TRAVIS_TAG) -X main.GitCommit=$(TRAVIS_COMMIT) -X main.GitRepoSlug=$(TRAVIS_REPO_SLUG) -X main.BuildDate=`date -u +%Y%m%d.%H%M%S`" github.com/kabanero-io/kabanero-operator/cmd/manager/collection
	GO111MODULE=on CGO_ENABLED=0 GOOS=linux GOARCH=$(ARCH) go build -o build/_output/bin/kabanero-operator-stack-controller -gcflags "all=-trimpath=$(GOPATH)" -asmflags "all=-trimpath=$(GOPATH)" -ldflags "-X main.GitTag=$(TRAVIS_TAG) -X main.GitCommit=$(TRAVIS_COMMIT) -X main.GitRepoSlug=$(TRAVIS_REPO_SLUG) -X main.BuildDate=`date -u +%Y%m%d.%H%M%S`" github.com/kabanero-io/kabanero-operator/cmd/manager/stack
	GO111MODULE=on CGO_ENABLED=0 GOOS=linux GOARCH=$(ARCH) go build -o build/_output/bin/admission-webhook -gcflags "all=-trimpath=$(GOPATH)" -asmflags "all=-trimpath=$(GOPATH)" -ldflags "-X main.GitTag=$(TRAVIS_TAG) -X main.GitCommit=$(TRAVIS_COMMIT) -X main.GitRepoSlug=$(TRAVIS_REPO_SLUG) -X main.BuildDate=`date -u +%Y%m%d.%H%M%S`" github.com/kabanero-io/kabanero-operator/cmd/admission-webhook

	docker build -f build/Dockerfile -t $(IMAGE) .

build-registry-image:
  # Build an OLM private registry for Kabanero. Should be run after push-image so the IMAGE SHA is generated
	mkdir -p build/registry
	cp LICENSE build/registry/LICENSE
	cp -R registry/manifests build/registry/
	cp registry/Dockerfile build/registry/Dockerfile
	cp deploy/crds/kabanero.io_kabaneros_crd.yaml deploy/crds/kabanero.io_collections_crd.yaml deploy/crds/kabanero.io_stacks_crd.yaml build/registry/manifests/kabanero-operator/$(CURRENT_RELEASE)/

# Use the internal service address in the CSV
ifdef INTERNAL_REGISTRY
	sed -e "s!kabanero/kabanero-operator:.*!$(REPOSITORY_SVC)@$(IMAGE_SHA)!" registry/manifests/kabanero-operator/$(CURRENT_RELEASE)/kabanero-operator.v$(CURRENT_RELEASE).clusterserviceversion.yaml > build/registry/manifests/kabanero-operator/$(CURRENT_RELEASE)/kabanero-operator.v$(CURRENT_RELEASE).clusterserviceversion.yaml
else
	sed -e "s!kabanero/kabanero-operator:.*!$(IMAGE_REPO_DIGEST)!" registry/manifests/kabanero-operator/$(CURRENT_RELEASE)/kabanero-operator.v$(CURRENT_RELEASE).clusterserviceversion.yaml > build/registry/manifests/kabanero-operator/$(CURRENT_RELEASE)/kabanero-operator.v$(CURRENT_RELEASE).clusterserviceversion.yaml
endif

	docker build -t $(REGISTRY_IMAGE) -f build/registry/Dockerfile build/registry/

	rm -R build/registry

push-image:
ifneq "$(IMAGE)" "kabanero-operator:latest"
  # Default push.  Make sure the namespace is there in case using local registry
	kubectl create namespace kabanero || true
	docker push $(IMAGE)
	# Get the RepoDigests
	docker pull $(IMAGE)
endif


push-registry-image:
ifneq "$(REGISTRY_IMAGE)" "kabanero-operator-registry:latest"
  # Default push.  Make sure the namespace is there in case using local registry
	kubectl create namespace kabanero || true
	docker push $(REGISTRY_IMAGE)
	# Get the RepoDigests
	docker pull $(REGISTRY_IMAGE)
endif


push-manifest:
	echo "IMAGE="$(IMAGE)
	docker manifest create $(IMAGE) $(IMAGE)-amd64 $(IMAGE)-ppc64le $(IMAGE)-s390x
	docker manifest annotate $(IMAGE) $(IMAGE)-amd64   --os linux --arch amd64
	docker manifest annotate $(IMAGE) $(IMAGE)-ppc64le --os linux --arch ppc64le
	docker manifest annotate $(IMAGE) $(IMAGE)-s390x   --os linux --arch s390x
	docker manifest inspect $(IMAGE)
	docker manifest push $(IMAGE) -p
	echo "REGISTRY_IMAGE="$(REGISTRY_IMAGE)
	docker manifest create $(REGISTRY_IMAGE) $(REGISTRY_IMAGE)-amd64 $(REGISTRY_IMAGE)-ppc64le $(REGISTRY_IMAGE)-s390x 
	docker manifest annotate $(REGISTRY_IMAGE) $(REGISTRY_IMAGE)-amd64   --os linux --arch amd64
	docker manifest annotate $(REGISTRY_IMAGE) $(REGISTRY_IMAGE)-ppc64le --os linux --arch ppc64le
	docker manifest annotate $(REGISTRY_IMAGE) $(REGISTRY_IMAGE)-s390x   --os linux --arch s390x
	docker manifest inspect $(REGISTRY_IMAGE)
	docker manifest push $(REGISTRY_IMAGE) -p

test: 
	GO111MODULE=on go test -cover ./cmd/... ./pkg/... 

format:
	GO111MODULE=on go fmt ./cmd/... ./pkg/...

generate:
	GO111MODULE=on operator-sdk generate k8s
	# GO111MODULE=on operator-sdk generate openapi
	GO111MODULE=on operator-sdk generate crds
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
	sed -i.bak -e 's!image: kabanero-operator:latest!image: $(IMAGE)!' deploy/operator.yaml
endif
	rm deploy/operator.yaml.bak || true
	kubectl config set-context $$(kubectl config current-context) --namespace=kabanero
	kubectl apply -f deploy/

deploy-olm:
	kubectl create namespace kabanero || true

	kubectl apply -f deploy/olm/

# Update deployment to correct image 
ifdef INTERNAL_REGISTRY
	sed -e "s!image: KABANERO_REGISTRY_IMAGE!image: $(REGISTRY_IMAGE_SVC)!" deploy/olm/01-catalog-source.yaml | kubectl apply -f -
else
	sed -e "s!image: KABANERO_REGISTRY_IMAGE!image: $(REGISTRY_IMAGE)!" deploy/olm/01-catalog-source.yaml | kubectl apply -f -
endif

check: format build test


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
	build/bin/golicense -plain ./license-rules.json build/_output/bin/admission-webhook build/_output/bin/kabanero-operator build/_output/bin/kabanero-operator-collection-controller build/_output/bin/kabanero-operator-stack-controller | sort > 3RD_PARTY || true
	rm build/bin/golicense

# Integration Tests
# Requires kube login context an existing cluster
# Requires internal registry with default route
# Requires jq

# Install Test
int-test-install: creds build-image push-image build-registry-image push-registry-image int-install int-config

creds:
	tests/00-credentials.sh

int-install:
# Update deployment to correct image 
ifdef INTERNAL_REGISTRY
	sed -e "s!image: kabanero/kabanero-operator-registry:.*!image: $(REPOSITORY_REGISTRY_SVC)@$(REGISTRY_IMAGE_SHA)!" deploy/kabanero-subscriptions.yaml > /tmp/kabanero-subscriptions.yaml
else
	sed -e "s!image: kabanero/kabanero-operator-registry:.*!image: $(REGISTRY_IMAGE_REPO_DIGEST)!" deploy/kabanero-subscriptions.yaml > /tmp/kabanero-subscriptions.yaml
endif

	KABANERO_SUBSCRIPTIONS_YAML=/tmp/kabanero-subscriptions.yaml KABANERO_CUSTOMRESOURCES_YAML=deploy/kabanero-customresources.yaml deploy/install.sh

# Rebuild & Reinstall only the kabanero-operator
int-test-kop-reinstall: creds build-image push-image build-registry-image push-registry-image int-kop-resub int-config

# Helper for int-op-reinstall
int-kop-resub:
	# Clean up instance, sub & csv
	kubectl -n kabanero delete kabanero kabanero --ignore-not-found=true && \
	CSV=$$(kubectl -n kabanero get subscription kabanero-operator --output=jsonpath={.status.installedCSV}) && \
	kubectl -n kabanero delete subscription kabanero-operator --ignore-not-found=true && \
	kubectl -n kabanero delete clusterserviceversion $${CSV} || true
	# Force a new catalog pod
	kubectl -n openshift-marketplace delete catalogsource kabanero-catalog --ignore-not-found=true
ifdef INTERNAL_REGISTRY
	sed -e "s!image: kabanero/kabanero-operator-registry:.*!image: $(REPOSITORY_REGISTRY_SVC)@$(REGISTRY_IMAGE_SHA)!" deploy/kabanero-subscriptions.yaml | kubectl -n openshift-marketplace apply --selector kabanero.io/install=00-catalogsource -f -
else 
	sed -e "s!image: kabanero/kabanero-operator-registry:.*!image: $(REGISTRY_IMAGE_REPO_DIGEST)!"  deploy/kabanero-subscriptions.yaml | kubectl -n openshift-marketplace apply --selector kabanero.io/install=00-catalogsource -f -
endif
	kubectl -n kabanero apply --selector kabanero.io/install=14-subscription -f deploy/kabanero-subscriptions.yaml 



int-config:
# Update config to correct image
ifdef INTERNAL_REGISTRY
	sed -e "s!image: kabanero/kabanero-operator:.*!image: $(REPOSITORY_SVC)@$(IMAGE_SHA)!" config/samples/full.yaml | kubectl -n kabanero apply -f -
else
	sed -e "s!image: kabanero/kabanero-operator:.*!image: $(IMAGE_REPO_DIGEST)!" config/samples/full.yaml | kubectl -n kabanero apply -f -
endif

# Uninstall Test
int-test-uninstall: creds int-uninstall

int-uninstall:
	KABANERO_SUBSCRIPTIONS_YAML=deploy/kabanero-subscriptions.yaml KABANERO_CUSTOMRESOURCES_YAML=deploy/kabanero-customresources.yaml deploy/uninstall.sh

# Stacks: Can be run in parallel ( -j ). Test manual pipeline run of stacks.
int-test-stacks: int-test-java-microprofile int-test-java-spring-boot2 int-test-nodejs int-test-nodejs-express int-test-java-openliberty

int-test-java-microprofile:
	tests/10-java-microprofile.sh

int-test-java-spring-boot2:
	tests/11-java-spring-boot2.sh

int-test-nodejs:
	tests/12-nodejs.sh

int-test-nodejs-express:
	tests/13-nodejs-express.sh

int-test-java-openliberty:
	tests/14-java-openliberty.sh

# Lifecycle: Lifecycle of Kabanero, Stack and owned objects
int-test-lifecycle: int-test-delete-pipeline int-test-delete-stack int-test-update-index int-test-deactivate-stack
int-test-delete-pipeline:
	tests/20-delete-pipeline.sh
int-test-delete-stack:
	tests/21-delete-stack.sh
int-test-update-index:
	tests/22-update-index.sh
int-test-deactivate-stack:
	tests/23-deactivate-stack.sh
