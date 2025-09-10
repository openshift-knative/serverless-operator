FROM registry.ci.openshift.org/origin/scos-4.20:operator-registry AS opm

FROM registry.access.redhat.com/ubi9/ubi-minimal as builder

COPY --from=opm /bin/opm /bin/opm

# Copy declarative config root into image at /configs
COPY olm-catalog/serverless-operator-index/configs /configs

# Copy policy.json for opm
COPY olm-catalog/serverless-operator-index/policy.json /etc/containers/policy.json

RUN /bin/opm init serverless-operator --default-channel=stable --output yaml >> /configs/index.yaml
RUN /bin/opm render --skip-tls-verify -o yaml \
registry.ci.openshift.org/knative/release-1.35.0:serverless-bundle \
registry.ci.openshift.org/knative/release-1.36.0:serverless-bundle \
      quay.io/redhat-user-workloads/ocp-serverless-tenant/serverless-operator-136/serverless-bundle@sha256:1d3861c2c8b0cf8c7aa747fe6503fa80b53727dc5adc9e4e643d4d3421bb6bef >> /configs/index.yaml

# The base image is expected to contain
# /bin/opm (with a serve subcommand) and /bin/grpc_health_probe
FROM registry.ci.openshift.org/origin/scos-4.20:operator-registry

# Copy declarative config root into image at /configs
COPY --from=builder /configs /configs

# Set DC-specific label for the location of the DC root directory
# in the image
LABEL operators.operatorframework.io.index.configs.v1=/configs

# Configure the entrypoint and command
ENTRYPOINT ["/bin/opm"]
CMD ["serve", "/configs"]
