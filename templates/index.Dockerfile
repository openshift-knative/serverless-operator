FROM quay.io/openshift/origin-operator-registry:4.11 AS opm

FROM quay.io/openshift/origin-base:4.11 as builder

COPY --from=opm /bin/opm /bin/opm

# Copy declarative config root into image at /configs
COPY configs /configs

RUN /bin/opm init serverless-operator --default-channel=stable --output yaml >> /configs/index.yaml
RUN /bin/opm render --skip-tls-verify -o yaml registry.ci.openshift.org/knative/openshift-serverless-v__PREVIOUS_VERSION__:serverless-stop-bundle \
      registry.ci.openshift.org/knative/openshift-serverless-v__VERSION_MAJOR_MINOR__.0:serverless-bundle >> /configs/index.yaml || \
    /bin/opm render --skip-tls-verify -o yaml registry.ci.openshift.org/knative/openshift-serverless-v__PREVIOUS_VERSION__:serverless-stop-bundle \
      registry.ci.openshift.org/knative/openshift-serverless-nightly:serverless-bundle >> /configs/index.yaml

# The base image is expected to contain
# /bin/opm (with a serve subcommand) and /bin/grpc_health_probe
FROM opm

# Copy declarative config root into image at /configs
COPY --from=builder /configs /configs

# Set DC-specific label for the location of the DC root directory
# in the image
LABEL operators.operatorframework.io.index.configs.v1=/configs

# Configure the entrypoint and command
ENTRYPOINT ["/bin/opm"]
CMD ["serve", "/configs"]
