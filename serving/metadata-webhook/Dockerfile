# DO NOT EDIT! Generated Dockerfile for serving/metadata-webhook/cmd/webhook.
ARG GO_BUILDER=registry.ci.openshift.org/openshift/release:rhel-8-release-golang-1.22-openshift-4.17
ARG GO_RUNTIME=registry.access.redhat.com/ubi8/ubi-minimal

FROM $GO_BUILDER as builder

COPY . .

ENV CGO_ENABLED=1
ENV GOEXPERIMENT=strictfipsruntime

RUN go build -tags strictfipsruntime -o /usr/bin/main ./serving/metadata-webhook/cmd/webhook

FROM $GO_RUNTIME

ARG VERSION=

COPY --from=builder /usr/bin/main /usr/bin/webhook

USER 65532

LABEL \
      com.redhat.component="openshift-serverless-1-webhook-rhel8-container" \
      name="openshift-serverless-1/webhook-rhel8" \
      version=$VERSION \
      summary="Red Hat OpenShift Serverless 1 Webhook" \
      maintainer="serverless-support@redhat.com" \
      description="Red Hat OpenShift Serverless 1 Webhook" \
      io.k8s.display-name="Red Hat OpenShift Serverless 1 Webhook" \
      io.k8s.description="Red Hat OpenShift Serverless Webhook" \
      io.openshift.tags="webhook"

ENTRYPOINT ["/usr/bin/webhook"]
