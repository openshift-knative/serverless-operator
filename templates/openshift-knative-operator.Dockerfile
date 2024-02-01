FROM registry.ci.openshift.org/openshift/release:rhel-8-release-golang-__GOLANG_VERSION__-openshift-4.16 AS builder

ENV BASE=github.com/openshift-knative/serverless-operator
WORKDIR ${GOPATH}/src/${BASE}

COPY . .

ENV GOFLAGS="-mod=vendor"
RUN go build -o /tmp/operator ${BASE}/openshift-knative-operator/cmd/operator

FROM registry.ci.openshift.org/ocp/ubi-minimal:8
USER 65532

COPY --from=builder /tmp/operator /ko-app/operator

ENV KO_DATA_PATH="/var/run/ko"
COPY openshift-knative-operator/cmd/operator/kodata $KO_DATA_PATH

ENTRYPOINT ["/ko-app/operator"]
