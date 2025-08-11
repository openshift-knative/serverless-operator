FROM registry.ci.openshift.org/origin/scos-4.18:operator-registry AS opm

FROM registry.access.redhat.com/ubi9/ubi-minimal as builder

COPY --from=opm /bin/opm /bin/opm

# Copy declarative config root into image at /configs
COPY olm-catalog/serverless-operator-index/configs /configs

# Copy policy.json for opm
COPY olm-catalog/serverless-operator-index/policy.json /etc/containers/policy.json

RUN /bin/opm init serverless-operator --default-channel=stable-1.35 --output yaml >> /configs/index.yaml
RUN /bin/opm render --skip-tls-verify -o yaml \
registry.redhat.io/openshift-serverless-1/serverless-operator-bundle:1.33.0 \
registry.redhat.io/openshift-serverless-1/serverless-operator-bundle:1.34.0 \
registry.redhat.io/openshift-serverless-1/serverless-operator-bundle:1.35.0 \
registry.redhat.io/openshift-serverless-1/serverless-operator-bundle:1.35.1 \
      registry.ci.openshift.org/knative/serverless-bundle@sha256:794a8a52da82f11da7a1a7727a283241fe2a2e4a2ef73f6fde55839ea737136b >> /configs/index.yaml

# The base image is expected to contain
# /bin/opm (with a serve subcommand) and /bin/grpc_health_probe
FROM registry.ci.openshift.org/origin/scos-4.18:operator-registry

# Copy declarative config root into image at /configs
COPY --from=builder /configs /configs

# Set DC-specific label for the location of the DC root directory
# in the image
LABEL operators.operatorframework.io.index.configs.v1=/configs

# Configure the entrypoint and command
ENTRYPOINT ["/bin/opm"]
CMD ["serve", "/configs"]
