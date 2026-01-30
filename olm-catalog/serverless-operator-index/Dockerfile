FROM registry.ci.openshift.org/origin/scos-4.22:operator-registry AS opm

FROM registry.access.redhat.com/ubi9/ubi-minimal as builder

COPY --from=opm /bin/opm /bin/opm

# Copy declarative config root into image at /configs
COPY olm-catalog/serverless-operator-index/configs /configs

# Copy policy.json for opm
COPY olm-catalog/serverless-operator-index/policy.json /etc/containers/policy.json

# Copy pre-generated yaml of bundles of the previous releases
COPY olm-catalog/serverless-operator-index/index-bundles.yaml /index-bundles.yaml

RUN /bin/opm init serverless-operator --default-channel=stable --output yaml >> /configs/index.yaml
RUN cat /index-bundles.yaml >> /configs/index.yaml
RUN /bin/opm render --skip-tls-verify -o yaml \
      quay.io/redhat-user-workloads/ocp-serverless-tenant/serverless-operator-137/serverless-bundle@sha256:060a5f8cd63f5d46bc714dc788f679cc89ab5beb5c0b288918f5002e0640ff18 >> /configs/index.yaml

# The base image is expected to contain
# /bin/opm (with a serve subcommand) and /bin/grpc_health_probe
FROM registry.ci.openshift.org/origin/scos-4.22:operator-registry

# Copy declarative config root into image at /configs
COPY --from=builder /configs /configs

# Set DC-specific label for the location of the DC root directory
# in the image
LABEL operators.operatorframework.io.index.configs.v1=/configs

# Configure the entrypoint and command
ENTRYPOINT ["/bin/opm"]
CMD ["serve", "/configs"]
