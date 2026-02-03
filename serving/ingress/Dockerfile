# DO NOT EDIT! Generated Dockerfile for serving/ingress/cmd/ingress.
ARG GO_BUILDER=registry.ci.openshift.org/openshift/release:rhel-9-release-golang-1.24-openshift-4.20
ARG GO_RUNTIME=registry.access.redhat.com/ubi9/ubi-minimal

FROM $GO_BUILDER as builder

WORKDIR /workspace
COPY . .

ENV CGO_ENABLED=1
ENV GOEXPERIMENT=strictfipsruntime

RUN go build -tags strictfipsruntime -o /usr/bin/main ./serving/ingress/cmd/ingress

FROM $GO_RUNTIME

ARG VERSION=

COPY --from=builder /usr/bin/main /usr/bin/ingress
COPY LICENSE /licenses/

USER 65532

LABEL \
      com.redhat.component="openshift-serverless-1-serverless-ingress-rhel9-container" \
      name="openshift-serverless-1/serverless-ingress-rhel9" \
      version=$VERSION \
      summary="Red Hat OpenShift Serverless 1 Ingress" \
      maintainer="serverless-support@redhat.com" \
      description="Red Hat OpenShift Serverless 1 Ingress" \
      io.k8s.display-name="Red Hat OpenShift Serverless 1 Ingress" \
      io.k8s.description="Red Hat OpenShift Serverless Ingress" \
      io.openshift.tags="ingress" \
      vendor="Red Hat, Inc." \
      release=$VERSION \
      cpe="cpe:/a:redhat:openshift_serverless:1.37::el9"

ENTRYPOINT ["/usr/bin/ingress"]
