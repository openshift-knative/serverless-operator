# DO NOT EDIT! Generated Dockerfile for serving/metadata-webhook/cmd/webhook.
ARG GO_BUILDER=registry.ci.openshift.org/openshift/release:rhel-9-release-golang-1.23-openshift-4.19
ARG GO_RUNTIME=registry.access.redhat.com/ubi9/ubi-minimal

FROM $GO_BUILDER as builder

WORKDIR /workspace
COPY . .

ENV CGO_ENABLED=1
ENV GOEXPERIMENT=strictfipsruntime

RUN go build -tags strictfipsruntime -o /usr/bin/main ./serving/metadata-webhook/cmd/webhook

FROM $GO_RUNTIME

ARG VERSION=

COPY --from=builder /usr/bin/main /usr/bin/webhook
COPY LICENSE /licenses/

USER 65532

LABEL \
      com.redhat.component="openshift-serverless-1-webhook-rhel9-container" \
      name="openshift-serverless-1/webhook-rhel9" \
      version=$VERSION \
      summary="Red Hat OpenShift Serverless 1 Webhook" \
      maintainer="serverless-support@redhat.com" \
      description="Red Hat OpenShift Serverless 1 Webhook" \
      io.k8s.display-name="Red Hat OpenShift Serverless 1 Webhook" \
      io.k8s.description="Red Hat OpenShift Serverless Webhook" \
      io.openshift.tags="webhook" \
      vendor="Red Hat, Inc." \
      release=$VERSION \
      cpe="cpe:/a:redhat:openshift_serverless:1.37::el9"

ENTRYPOINT ["/usr/bin/webhook"]
