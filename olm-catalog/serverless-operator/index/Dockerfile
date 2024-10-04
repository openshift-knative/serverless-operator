FROM registry.ci.openshift.org/origin/4.17:operator-registry AS opm

FROM registry.access.redhat.com/ubi9/ubi-minimal as builder

COPY --from=opm /bin/opm /bin/opm

# Copy declarative config root into image at /configs
COPY olm-catalog/serverless-operator/index/configs /configs

RUN /bin/opm init serverless-operator --default-channel=stable --output yaml >> /configs/index.yaml
RUN /bin/opm render --skip-tls-verify -o yaml \
registry.ci.openshift.org/knative/release-1.33.0:serverless-bundle \
registry.ci.openshift.org/knative/release-1.34.0:serverless-bundle \
      registry.ci.openshift.org/knative/release-1.35.0:serverless-bundle >> /configs/index.yaml || \
    /bin/opm render --skip-tls-verify -o yaml \
registry.ci.openshift.org/knative/release-1.33.0:serverless-bundle \
registry.ci.openshift.org/knative/release-1.34.0:serverless-bundle \
      registry.redhat.io/openshift-serverless-1/serverless-bundle@sha256:7ba30804bd6cc2c2206919568446b00f41c518849c00328159112c80546cfde9 >> /configs/index.yaml

# The base image is expected to contain
# /bin/opm (with a serve subcommand) and /bin/grpc_health_probe
FROM registry.ci.openshift.org/origin/4.17:operator-registry

# Copy declarative config root into image at /configs
COPY --from=builder /configs /configs

# Set DC-specific label for the location of the DC root directory
# in the image
LABEL operators.operatorframework.io.index.configs.v1=/configs

# Configure the entrypoint and command
ENTRYPOINT ["/bin/opm"]
CMD ["serve", "/configs"]
