FROM alpine

COPY manifests /manifests
COPY metadata/annotations.yaml /metadata/annotations.yaml

LABEL operators.operatorframework.io.bundle.mediatype.v1=registry+v1
LABEL operators.operatorframework.io.bundle.manifests.v1=manifests/
LABEL operators.operatorframework.io.bundle.metadata.v1=metadata/
LABEL operators.operatorframework.io.bundle.package.v1=__NAME__
LABEL operators.operatorframework.io.bundle.channel.default.v1="__DEFAULT_CHANNEL__"
LABEL operators.operatorframework.io.bundle.channels.v1="__CHANNEL_LIST__"

LABEL \
      com.redhat.component="openshift-serverless-1-serverless-operator-bundle-container" \
      name="openshift-serverless-1/serverless-operator-bundle" \
      version="__VERSION__" \
      summary="Red Hat OpenShift Serverless Bundle" \
      maintainer="serverless-support@redhat.com" \
      description="Red Hat OpenShift Serverless Bundle" \
      io.k8s.display-name="Red Hat OpenShift Serverless Bundle" \
      com.redhat.openshift.versions="__OCP_TARGET_VLIST__" \
      com.redhat.delivery.operator.bundle=true \
      com.redhat.delivery.backport=false

# Remove the "replaces" line to make this bundle be able to be "the last".
RUN sed -i '/replaces:/d' /manifests/serverless-operator.clusterserviceversion.yaml
