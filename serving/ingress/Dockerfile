# DO NOT EDIT! Generated Dockerfile for serving/ingress/cmd/ingress.
ARG GO_BUILDER=registry.ci.openshift.org/openshift/release:rhel-8-release-golang-1.21-openshift-4.17
ARG GO_RUNTIME=registry.access.redhat.com/ubi8/ubi-minimal

FROM $GO_BUILDER as builder

COPY . .

ENV CGO_ENABLED=1
ENV GOEXPERIMENT=strictfipsruntime

RUN go build -tags strictfipsruntime -o /usr/bin/main ./serving/ingress/cmd/ingress

FROM $GO_RUNTIME

ARG VERSION=

RUN microdnf install tzdata 

USER 65532

COPY --from=builder /usr/bin/main /usr/bin/main

LABEL \
      com.redhat.component="openshift-serverless-1-ingress-rhel8-container" \
      name="openshift-serverless-1/ingress-rhel8" \
      version=$VERSION \
      summary="Red Hat OpenShift Serverless 1 Ingress" \
      maintainer="serverless-support@redhat.com" \
      description="Red Hat OpenShift Serverless 1 Ingress" \
      io.k8s.display-name="Red Hat OpenShift Serverless 1 Ingress"

ENTRYPOINT ["/usr/bin/main"]
