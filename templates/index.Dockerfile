FROM registry.redhat.io/openshift4/ose-operator-registry-rhel9:v__OCP_MAX_VERSION__ AS opm

FROM registry.access.redhat.com/ubi9/ubi-minimal as builder

COPY --from=opm /bin/opm /bin/opm

# Copy declarative config root into image at /configs
COPY olm-catalog/serverless-operator/index/configs /configs

RUN /bin/opm init serverless-operator --default-channel=__DEFAULT_CHANNEL__ --output yaml >> /configs/index.yaml
RUN /bin/opm render --skip-tls-verify -o yaml \
      registry.ci.openshift.org/knative/release-__VERSION__:serverless-bundle >> /configs/index.yaml || \
    /bin/opm render --skip-tls-verify -o yaml \
      __BUNDLE__ >> /configs/index.yaml

# The base image is expected to contain
# /bin/opm (with a serve subcommand) and /bin/grpc_health_probe
FROM registry.redhat.io/openshift4/ose-operator-registry-rhel9:v__OCP_MAX_VERSION__

# Copy declarative config root into image at /configs
COPY --from=builder /configs /configs

# Set DC-specific label for the location of the DC root directory
# in the image
LABEL operators.operatorframework.io.index.configs.v1=/configs

# Configure the entrypoint and command
ENTRYPOINT ["/bin/opm"]
CMD ["serve", "/configs"]
