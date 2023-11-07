FROM registry.ci.openshift.org/openshift/release:golang-__GOLANG_VERSION__ AS builder

ENV BASE=github.com/openshift-knative/serverless-operator
WORKDIR ${GOPATH}/src/${BASE}

COPY . .

ENV GOFLAGS="-mod=vendor"
RUN go build -o /tmp/operator ${BASE}/openshift-knative-operator/cmd/operator

FROM registry.ci.openshift.org/ocp/__OCP_MAX_VERSION__:base
USER 65532

COPY --from=builder /tmp/operator /ko-app/operator

ENV KO_DATA_PATH="/var/run/ko"
COPY openshift-knative-operator/cmd/operator/kodata $KO_DATA_PATH

ENTRYPOINT ["/ko-app/operator"]
