# DO NOT EDIT! Generated Dockerfile for openshift-knative-operator/cmd/openshift-knative-operator.
ARG GO_BUILDER=registry.ci.openshift.org/openshift/release:rhel-9-release-golang-1.24-openshift-4.20
ARG GO_RUNTIME=registry.access.redhat.com/ubi9/ubi-minimal

FROM $GO_BUILDER as builder

WORKDIR /workspace
COPY . .

ENV CGO_ENABLED=1
ENV GOEXPERIMENT=strictfipsruntime

RUN go build -tags strictfipsruntime -o /usr/bin/main ./openshift-knative-operator/cmd/openshift-knative-operator

FROM $GO_RUNTIME

ARG VERSION=

COPY --from=builder /usr/bin/main /usr/bin/openshift-knative-operator
COPY LICENSE /licenses/

ENV KO_DATA_PATH="/var/run/ko"
COPY openshift-knative-operator/cmd/openshift-knative-operator/kodata $KO_DATA_PATH

USER 65532

LABEL \
      com.redhat.component="openshift-serverless-1-serverless-openshift-kn-rhel9-operator-container" \
      name="openshift-serverless-1/serverless-openshift-kn-rhel9-operator" \
      version=$VERSION \
      summary="Red Hat OpenShift Serverless 1 Openshift Knative Operator" \
      maintainer="serverless-support@redhat.com" \
      description="Red Hat OpenShift Serverless 1 Openshift Knative Operator" \
      io.k8s.display-name="Red Hat OpenShift Serverless 1 Openshift Knative Operator" \
      io.k8s.description="Red Hat OpenShift Serverless Openshift Knative Operator" \
      io.openshift.tags="openshift-knative-operator" \
      vendor="Red Hat, Inc." \
      release=$VERSION \
      cpe="cpe:/a:redhat:openshift_serverless:1.37::el9"

ENTRYPOINT ["/usr/bin/openshift-knative-operator"]
