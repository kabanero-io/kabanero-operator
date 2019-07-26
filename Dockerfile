# This is a workaround until manfistival can interact with the virtual file system
ARG IMAGE=kabanero:latest

FROM ${IMAGE}

COPY config /config