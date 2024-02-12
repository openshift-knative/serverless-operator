FROM registry.ci.openshift.org/openshift/release:rhel-8-release-golang-1.21-openshift-4.16 AS builder

ENV BASE=github.com/openshift-knative/serverless-operator
WORKDIR ${GOPATH}/src/${BASE}

COPY . .

ENV GOFLAGS="-mod=vendor"
RUN go build -o /tmp/metadata-webhook ${BASE}/serving/metadata-webhook/cmd/webhook

FROM registry.ci.openshift.org/ocp/ubi-minimal:8
USER 65532

COPY --from=builder /tmp/metadata-webhook /ko-app/metadata-webhook

ENTRYPOINT ["/ko-app/metadata-webhook"]
