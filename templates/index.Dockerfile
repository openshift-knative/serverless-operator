FROM openshift/origin-base as builder

COPY --from=quay.io/operator-framework/opm:latest /bin/opm /bin/opm

# Copy declarative config root into image at /configs
COPY configs /configs

RUN /bin/opm init serverless-operator --default-channel=stable --output yaml >> /configs/index.yaml &&
  /bin/opm --skip-tls-verify render -o yaml registry.ci.openshift.org/openshift/openshift-serverless-v__PREVIOUS_VERSION__:serverless-stop-bundle \
  image-registry.openshift-image-registry.svc:5000/openshift-marketplace/serverless-bundle:latest >> /configs/index.yaml

# The base image is expected to contain
# /bin/opm (with a serve subcommand) and /bin/grpc_health_probe
FROM quay.io/operator-framework/opm:latest

# Copy declarative config root into image at /configs
COPY --from=builder /configs /configs

# Set DC-specific label for the location of the DC root directory
# in the image
LABEL operators.operatorframework.io.index.configs.v1=/configs

# Configure the entrypoint and command
ENTRYPOINT ["/bin/opm"]
CMD ["serve", "/configs"]
