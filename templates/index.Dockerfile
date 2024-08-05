FROM registry.ci.openshift.org/origin/__OCP_MAX_VERSION__:operator-registry AS opm

FROM registry.access.redhat.com/ubi9/ubi-minimal as builder

COPY --from=opm /bin/opm /bin/opm

# Copy declarative config root into image at /configs
COPY olm-catalog/serverless-operator/index/configs /configs

RUN /bin/opm init serverless-operator --default-channel=__DEFAULT_CHANNEL__ --output yaml >> /configs/index.yaml
# Workaround for https://issues.redhat.com/browse/SRVCOM-3207
# Use a manually built image for serverless-bundle.
# TODO: Change to registry.ci.openshift.org when not using 1.32.0. This is a problem only for 1.32.0.
RUN /bin/opm render --skip-tls-verify -o yaml \
      __PREVIOUS_PREVIOUS_VERSION__ \
      quay.io/openshift-knative/serverless-bundle:release-__PREVIOUS_REPLACES__ \
      registry.ci.openshift.org/knative/release-__PREVIOUS_VERSION__:serverless-bundle \
      registry.ci.openshift.org/knative/release-__VERSION__:serverless-bundle >> /configs/index.yaml || \
    /bin/opm render --skip-tls-verify -o yaml \
      __PREVIOUS_PREVIOUS_VERSION__ \
      quay.io/openshift-knative/serverless-bundle:release-__PREVIOUS_REPLACES__ \
      registry.ci.openshift.org/knative/release-__PREVIOUS_VERSION__:serverless-bundle \
      registry.ci.openshift.org/knative/serverless-bundle:main >> /configs/index.yaml

# The base image is expected to contain
# /bin/opm (with a serve subcommand) and /bin/grpc_health_probe
FROM registry.ci.openshift.org/origin/__OCP_MAX_VERSION__:operator-registry

# Copy declarative config root into image at /configs
COPY --from=builder /configs /configs

# Set DC-specific label for the location of the DC root directory
# in the image
LABEL operators.operatorframework.io.index.configs.v1=/configs

# Configure the entrypoint and command
ENTRYPOINT ["/bin/opm"]
CMD ["serve", "/configs"]
