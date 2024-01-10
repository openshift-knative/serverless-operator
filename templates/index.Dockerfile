FROM registry.ci.openshift.org/origin/__OCP_MAX_VERSION__:operator-registry AS opm

FROM registry.ci.openshift.org/ocp/__OCP_MAX_VERSION__:base as builder

COPY --from=opm /bin/opm /bin/opm

# Copy declarative config root into image at /configs
COPY olm-catalog/serverless-operator/index/configs /configs

RUN /bin/opm init serverless-operator --default-channel=stable --output yaml >> /configs/index.yaml
RUN /bin/opm render --skip-tls-verify -o yaml registry.ci.openshift.org/knative/openshift-serverless-v__PREVIOUS_REPLACES__:serverless-bundle \
      registry.ci.openshift.org/knative/openshift-serverless-v__PREVIOUS_VERSION__:serverless-bundle \
      registry.ci.openshift.org/knative/openshift-serverless-v__VERSION__:serverless-bundle >> /configs/index.yaml || \
    /bin/opm render --skip-tls-verify -o yaml registry.ci.openshift.org/knative/openshift-serverless-v__PREVIOUS_REPLACES__:serverless-bundle \
      registry.ci.openshift.org/knative/openshift-serverless-v__PREVIOUS_VERSION__:serverless-bundle \
      registry.ci.openshift.org/knative/serverless-bundle:main >> /configs/index.yaml

# The base image is expected to contain
# /bin/opm (with a serve subcommand) and /bin/grpc_health_probe
FROM quay.io/openshift/origin-operator-registry:__OCP_MAX_VERSION__

# Copy declarative config root into image at /configs
COPY --from=builder /configs /configs

# Set DC-specific label for the location of the DC root directory
# in the image
LABEL operators.operatorframework.io.index.configs.v1=/configs

# Configure the entrypoint and command
ENTRYPOINT ["/bin/opm"]
CMD ["serve", "/configs"]
