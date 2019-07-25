# leave as is or define  as a travis-ci env variables
DOCKERHUB_PROJECT ?= kabanero
DOCKERHUB_REPO ?= kabanero-operator
# leave as is else this is overridden by github release tag
TRAVIS_BRANCH ?= latest
IMAGE ?= ${DOCKERHUB_PROJECT}/${DOCKERHUB_REPO}:${TRAVIS_BRANCH}

.PHONY: build deploy

build: dependencies
	go install ./cmd/manager

build-image: dependencies generate
	operator-sdk build ${IMAGE}
	# This is a workaround until manfistival can interact with the virtual file system
	docker build -t ${IMAGE} --build-arg DOCKERHUB_PROJECT=${DOCKERHUB_PROJECT} --build-arg DOCKERHUB_REPO=${DOCKERHUB_REPO} --build-arg TRAVIS_BRANCH=${TRAVIS_BRANCH} .

push-image:
	docker images
	# docker user/pass travis-ci env variables
	docker login -u $(DOCKER_USERNAME) -p $(DOCKER_PASSWORD)
	docker push $(IMAGE)

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