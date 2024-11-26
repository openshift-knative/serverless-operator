ARG GO_RUNTIME=registry.access.redhat.com/ubi8/ubi-minimal

FROM $GO_RUNTIME

COPY olm-catalog/serverless-operator/manifests /manifests
COPY olm-catalog/serverless-operator/metadata/annotations.yaml /metadata/annotations.yaml
COPY LICENSE /licenses/

USER 65532

LABEL operators.operatorframework.io.bundle.mediatype.v1=registry+v1
LABEL operators.operatorframework.io.bundle.manifests.v1=manifests/
LABEL operators.operatorframework.io.bundle.metadata.v1=metadata/
LABEL operators.operatorframework.io.bundle.package.v1=__NAME__
LABEL operators.operatorframework.io.bundle.channel.default.v1="__DEFAULT_CHANNEL__"
LABEL operators.operatorframework.io.bundle.channels.v1="__DEFAULT_CHANNEL__,__LATEST_VERSIONED_CHANNEL__"

LABEL \
      com.redhat.component="openshift-serverless-1-serverless-operator-bundle-container" \
      name="openshift-serverless-1/serverless-operator-bundle" \
      version="__VERSION__" \
      summary="Red Hat OpenShift Serverless Bundle" \
      maintainer="serverless-support@redhat.com" \
      description="Red Hat OpenShift Serverless Bundle" \
      io.k8s.description="Red Hat OpenShift Serverless Bundle" \
      io.k8s.display-name="Red Hat OpenShift Serverless Bundle" \
      com.redhat.openshift.versions="__OCP_TARGET_VLIST__" \
      com.redhat.delivery.operator.bundle=true \
      com.redhat.delivery.backport=false \
      distribution-scope="authoritative-source-only" \
      url="https://catalog.redhat.com/software/container-stacks/detail/5ec53fcb110f56bd24f2ddc5" \
      release="__VERSION__" \
      io.openshift.tags="bundle" \
      vendor="Red Hat, Inc."
