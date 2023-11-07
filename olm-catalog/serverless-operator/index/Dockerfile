FROM registry.ci.openshift.org/origin/4.13:operator-registry AS opm

FROM registry.ci.openshift.org/ocp/4.13:base as builder

COPY --from=opm /bin/opm /bin/opm

COPY . .
# Backwards compatible with CI config that specifies context_dir pointing to
# olm-catalog/serverless-operator/index dir.
RUN mv olm-catalog/serverless-operator/index/configs /configs || mv configs /configs

RUN /bin/opm init serverless-operator --default-channel=stable --output yaml >> /configs/index.yaml
RUN /bin/opm render --skip-tls-verify -o yaml registry.ci.openshift.org/knative/openshift-serverless-v1.30.0:serverless-bundle \
      registry.ci.openshift.org/knative/openshift-serverless-v1.31.0:serverless-bundle \
      registry.ci.openshift.org/knative/openshift-serverless-v1.32.0:serverless-bundle >> /configs/index.yaml || \
    /bin/opm render --skip-tls-verify -o yaml registry.ci.openshift.org/knative/openshift-serverless-v1.30.0:serverless-bundle \
      registry.ci.openshift.org/knative/openshift-serverless-v1.31.0:serverless-bundle \
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
