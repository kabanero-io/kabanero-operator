#!/bin/bash

set -Eeuox pipefail

COLLECTION="nodejs-express"
APP="sample-nodejs-express" \
DOCKER_IMAGE="image-registry.openshift-image-registry.svc:5000/kabanero/${APP}" \
APP_REPO="https://github.com/kabanero-io/${APP}/" \
PIPELINE_RUN="${APP}-build-deploy-pl-run-kabanero" \
PIPELINE_REF="${COLLECTION}-build-deploy-pl" \
DOCKER_IMAGE_REF="${APP}-docker-image" \
GITHUB_SOURCE_REF="${APP}-git-source" \
$(dirname "$0")/pipelinerun.sh
