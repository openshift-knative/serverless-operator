# DO NOT EDIT! Generated Dockerfile for knative-operator/cmd/knative-operator.
ARG GO_BUILDER=registry.ci.openshift.org/openshift/release:rhel-8-release-golang-1.21-openshift-4.17
ARG GO_RUNTIME=registry.access.redhat.com/ubi8/ubi-minimal

FROM $GO_BUILDER as builder

COPY . .

ENV CGO_ENABLED=1
ENV GOEXPERIMENT=strictfipsruntime

RUN go build -tags strictfipsruntime -o /usr/bin/main ./knative-operator/cmd/knative-operator

FROM $GO_RUNTIME

ARG VERSION=

COPY --from=builder /usr/bin/main /usr/bin/knative-operator

COPY knative-operator/deploy /deploy

USER 65532

LABEL \
      com.redhat.component="openshift-serverless-1-knative-operator-rhel8-container" \
      name="openshift-serverless-1/knative-operator-rhel8" \
      version=$VERSION \
      summary="Red Hat OpenShift Serverless 1 Knative Operator" \
      maintainer="serverless-support@redhat.com" \
      description="Red Hat OpenShift Serverless 1 Knative Operator" \
      io.k8s.display-name="Red Hat OpenShift Serverless 1 Knative Operator"

ENTRYPOINT ["/usr/bin/knative-operator"]
