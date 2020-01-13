#!/bin/bash

set -Eeuox pipefail

COLLECTION="nodejs"
APP="sample-nodejs" \
DOCKER_IMAGE="image-registry.openshift-image-registry.svc:5000/kabanero/${APP}" \
APP_REPO="https://github.com/kabanero-io/${APP}/" \
PIPELINE_RUN="${APP}-build-deploy-pipeline-run-kabanero" \
PIPELINE_REF="${COLLECTION}-build-push-deploy-pipeline" \
DOCKER_IMAGE_REF="${APP}-docker-image" \
GITHUB_SOURCE_REF="${APP}-git-source" \
NO_APP_CHECK="TRUE" \
$(dirname "$0")/pipelinerun.sh
