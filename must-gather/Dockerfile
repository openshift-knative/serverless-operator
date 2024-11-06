# DO NOT EDIT! Generated Dockerfile for must-gather.
ARG MUSTGATHER=quay.io/openshift/origin-must-gather:4.13
ARG RUNTIME=registry.access.redhat.com/ubi8/ubi-minimal
FROM $MUSTGATHER AS mustgather

FROM $RUNTIME

COPY --from=mustgather /usr/bin/oc /usr/bin/oc

# Copy all collection scripts to /usr/bin
COPY must-gather/bin/* /usr/bin/

RUN microdnf install -y rsync tar

LABEL \
      com.redhat.component="openshift-serverless-1-must-gather-rhel8-container" \
      name="openshift-serverless-1/svls-must-gather-rhel8" \
      version=1.35.0 \
      summary="Red Hat OpenShift Serverless 1 Must Gather" \
      maintainer="serverless-support@redhat.com" \
      description="Red Hat OpenShift Serverless 1 Must Gather" \
      io.k8s.display-name="Red Hat OpenShift Serverless 1 Must Gather" \
      io.k8s.description="Red Hat OpenShift Serverless Must Gather" \
      io.openshift.tags="must-gather"

ENTRYPOINT /usr/bin/gather
