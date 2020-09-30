FROM scratch

COPY manifests /manifests
COPY metadata/annotations.yaml /metadata/annotations.yaml

LABEL operators.operatorframework.io.bundle.mediatype.v1=registry+v1
LABEL operators.operatorframework.io.bundle.manifests.v1=manifests/
LABEL operators.operatorframework.io.bundle.metadata.v1=metadata/
LABEL operators.operatorframework.io.bundle.package.v1=__NAME__
LABEL operators.operatorframework.io.bundle.channel.default.v1="__DEFAULT_CHANNEL__"
LABEL operators.operatorframework.io.bundle.channels.v1="__CHANNEL_LIST__"

LABEL \
      com.redhat.component="openshift-serverless-1-serverless-rhel8-operator-container" \
      name="openshift-serverless-1/serverless-rhel8-operator" \
      version="__VERSION__" \
      summary="Red Hat OpenShift Serverless 1 Serverless Operator" \
      maintainer="serverless-support@redhat.com" \
      description="Red Hat OpenShift Serverless 1 Serverless Operator" \
      io.k8s.display-name="Red Hat OpenShift Serverless 1 Serverless Operator" \
      com.redhat.openshift.versions="v4.5" \
      com.redhat.delivery.operator.bundle=true \
      com.redhat.delivery.backport=true
