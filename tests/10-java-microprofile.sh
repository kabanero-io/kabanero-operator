#!/bin/bash

set -Eeuox pipefail

DOCKER_IMAGE="image-registry.openshift-image-registry.svc:5000/kabanero/java-microprofile" \
APP_REPO="https://github.com/kabanero-io/sample-java-microprofile/" \
PIPELINE_RUN="java-microprofile-build-deploy-pipeline-run-kabanero" \
PIPELINE_REF="java-microprofile-build-push-deploy-pipeline" \
DOCKER_IMAGE_REF="java-microprofile-docker-image" \
GITHUB_SOURCE_REF="java-microprofile-git-source" \
$(dirname "$0")/pipelinerun.sh