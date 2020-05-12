FROM quay.io/operator-framework/upstream-registry-builder:v1.12.2 as builder

# This dockerfile is taken from the operator-registry upstream example
COPY manifests manifests
RUN ./bin/initializer -o ./bundles.db

FROM registry.access.redhat.com/ubi7/ubi-minimal:latest

# The following labels are required for Redhat container certification
LABEL vendor="Kabanero" \
      name="Kabanero Operator Registry" \
      summary="Image for Kabanero Operator Registry" \
      description="This image contains the registry that the Kabanero CatalogSource will read the Kabanero Operator from.  See https://github.com/kabanero-io/kabanero-operator/"

COPY --from=builder /build/bundles.db /bundles.db
COPY --from=builder /build/bin/registry-server /registry-server
COPY --from=builder /bin/grpc_health_probe /bin/grpc_health_probe

# The license must be here for Redhat container certification
COPY LICENSE /licenses/

EXPOSE 50051
ENTRYPOINT ["/registry-server"]
CMD ["--database", "bundles.db"]
