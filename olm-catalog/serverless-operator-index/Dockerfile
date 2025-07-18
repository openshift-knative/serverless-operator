FROM registry.ci.openshift.org/origin/scos-4.20:operator-registry AS opm

FROM registry.access.redhat.com/ubi9/ubi-minimal as builder

COPY --from=opm /bin/opm /bin/opm

# Copy declarative config root into image at /configs
COPY olm-catalog/serverless-operator-index/configs /configs

# Copy policy.json for opm
COPY olm-catalog/serverless-operator-index/policy.json /etc/containers/policy.json

RUN /bin/opm init serverless-operator --default-channel=stable --output yaml >> /configs/index.yaml
RUN /bin/opm render --skip-tls-verify -o yaml \
quay.io/redhat-user-workloads/ocp-serverless-tenant/serverless-operator-135/serverless-bundle:1.35.0 \
quay.io/redhat-user-workloads/ocp-serverless-tenant/serverless-operator-136/serverless-bundle:1.36.0 \
      quay.io/redhat-user-workloads/ocp-serverless-tenant/serverless-operator-137/serverless-bundle@sha256:c1ef1f98e9594a6d82f08c4db8c347a08e03bb30bde3be75ddfbdb7b24f65f14 >> /configs/index.yaml

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
