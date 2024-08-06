FROM registry.ci.openshift.org/openshift/release:rhel-8-release-golang-__GOLANG_VERSION__-openshift-4.17 AS builder

ENV BASE=github.com/openshift-knative/serverless-operator
WORKDIR ${GOPATH}/src/${BASE}

COPY . .

ENV GOFLAGS="-mod=vendor"
RUN go build -o /tmp/operator ${BASE}/serving/ingress/cmd/controller

FROM registry.access.redhat.com/ubi8-minimal:latest
USER 65532

COPY --from=builder /tmp/operator /ko-app/operator

ENTRYPOINT ["/ko-app/operator"]
