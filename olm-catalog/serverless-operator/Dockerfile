FROM alpine

COPY manifests /manifests
COPY metadata/annotations.yaml /metadata/annotations.yaml

# Substitute the to be deployed images based on the environment.
ARG repo
ENV DOCKER_REPO_OVERRIDE=$repo
RUN apk add gettext jq
RUN chmod +x /manifests/substitute.sh && /manifests/substitute.sh

LABEL operators.operatorframework.io.bundle.mediatype.v1=registry+v1
LABEL operators.operatorframework.io.bundle.manifests.v1=manifests/
LABEL operators.operatorframework.io.bundle.metadata.v1=metadata/
LABEL operators.operatorframework.io.bundle.package.v1=serverless-operator
LABEL operators.operatorframework.io.bundle.channel.default.v1="4.5"
LABEL operators.operatorframework.io.bundle.channels.v1="4.5"

LABEL \
      com.redhat.component="openshift-serverless-1-serverless-rhel8-operator-container" \
      name="openshift-serverless-1/serverless-rhel8-operator" \
      version="1.8.0" \
      summary="Red Hat OpenShift Serverless 1 Serverless Operator" \
      maintainer="serverless-support@redhat.com" \
      description="Red Hat OpenShift Serverless 1 Serverless Operator" \
      io.k8s.display-name="Red Hat OpenShift Serverless 1 Serverless Operator" \
      com.redhat.openshift.versions="v4.5" \
      com.redhat.delivery.operator.bundle=true \
      com.redhat.delivery.backport=true
