# DO NOT EDIT! Generated Dockerfile for must-gather.
ARG CLI_ARTIFACTS=registry.ci.openshift.org/ocp/4.16:cli-artifacts
ARG RUNTIME=registry.access.redhat.com/ubi9/ubi-minimal
FROM $CLI_ARTIFACTS AS cli-artifacts

FROM $RUNTIME

ARG TARGETARCH

COPY --from=cli-artifacts /usr/share/openshift/linux_$TARGETARCH/oc.rhel9 /usr/bin/oc

# Copy all collection scripts to /usr/bin
COPY must-gather/bin/* /usr/bin/
COPY LICENSE /licenses/

RUN microdnf install -y rsync tar

ENV LOGS_DIR="/must-gather"
RUN mkdir -p $LOGS_DIR && \
    chown -R 65532:65532 $LOGS_DIR

USER 65532

LABEL \
      com.redhat.component="openshift-serverless-1-must-gather--rhel9-container" \
      name="openshift-serverless-1/must-gather--rhel9" \
      version=1.37.0 \
      summary="Red Hat OpenShift Serverless 1 Must Gather" \
      maintainer="serverless-support@redhat.com" \
      description="Red Hat OpenShift Serverless 1 Must Gather" \
      io.k8s.display-name="Red Hat OpenShift Serverless 1 Must Gather" \
      io.k8s.description="Red Hat OpenShift Serverless Must Gather" \
      io.openshift.tags="must-gather" \
      vendor="Red Hat, Inc." \
      release=$VERSION \
      cpe="cpe:/a:redhat:openshift_serverless:1.37::el9"  

ENTRYPOINT /usr/bin/gather
