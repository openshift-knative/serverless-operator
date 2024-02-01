FROM registry.ci.openshift.org/openshift/release:rhel-8-release-golang-__GOLANG_VERSION__-openshift-4.16 AS builder

ENV BASE=github.com/openshift-knative/serverless-operator
WORKDIR ${GOPATH}/src/${BASE}

COPY . .

ENV GOFLAGS="-mod=vendor"
RUN go build -o /tmp/operator ${BASE}/knative-operator/cmd/manager

FROM registry.ci.openshift.org/ocp/ubi-minimal:8
USER 65532

COPY --from=builder /tmp/operator /ko-app/operator
# install manifest[s]
COPY knative-operator/deploy /deploy

ENTRYPOINT ["/ko-app/operator"]
