# This is a workaround until manfistival can interact with the virtual file system
ARG IMAGE=kabanero:latest

FROM ${IMAGE}

# The following labels are required for Redhat container certification
LABEL vendor="Kabanero" \
      name="Kabanero Operator" \
      summary="Image for Kabanaro Operator" \
      description="This image contains the controller for the Kabanero Foundation and Collection.  See https://github.com/kabanero-io/kabanero-operator/"

COPY config /config

# The licence must be here for Redhat container certification
COPY LICENSE /licenses/