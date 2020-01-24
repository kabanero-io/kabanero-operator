FROM registry.access.redhat.com/ubi7/ubi-minimal:latest

# The following labels are required for Redhat container certification
LABEL vendor="Kabanero" \
      name="Kabanero Operator" \
      summary="Image for Kabanero Operator" \
      description="This image contains the controller for the Kabanero Foundation and Collection.  See https://github.com/kabanero-io/kabanero-operator/"

# The license must be here for Redhat container certification
COPY LICENSE /licenses/

ENV OPERATOR=/usr/local/bin/kabanero-operator \
    USER_UID=1001 \
    USER_NAME=kabanero-operator

# install operator binary
COPY build/_output/bin/kabanero-operator ${OPERATOR}

COPY build/bin /usr/local/bin
RUN  /usr/local/bin/user_setup

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}
