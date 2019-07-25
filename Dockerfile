# This is a workaround until manfistival can interact with the virtual file system
ARG DOCKERHUB_PROJECT=kabanero
ARG DOCKERHUB_REPO=kabanero-operator
ARG TRAVIS_BRANCH=latest

FROM ${DOCKERHUB_PROJECT}/${DOCKERHUB_REPO}:${TRAVIS_BRANCH}

COPY config /config