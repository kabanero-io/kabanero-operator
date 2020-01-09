#!/bin/bash

set -Eeuox pipefail

COLLECTION="java-spring-boot2"
APP="sample-java-spring-boot2" \
DOCKER_IMAGE="image-registry.openshift-image-registry.svc:5000/kabanero/${APP}" \
APP_REPO="https://github.com/kabanero-io/${APP}/" \
PIPELINE_RUN="${APP}-build-deploy-pipeline-run-kabanero" \
PIPELINE_REF="${COLLECTION}-build-push-deploy-pipeline" \
DOCKER_IMAGE_REF="${APP}-docker-image" \
GITHUB_SOURCE_REF="${APP}-git-source" \
$(dirname "$0")/pipelinerun.sh
