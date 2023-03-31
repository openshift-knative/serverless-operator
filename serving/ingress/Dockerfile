FROM registry.ci.openshift.org/openshift/release:golang-1.18 AS builder

ENV BASE=github.com/openshift-knative/serverless-operator
WORKDIR ${GOPATH}/src/${BASE}

COPY . .

ENV GOFLAGS="-mod=vendor"
RUN go build -o /tmp/operator ${BASE}/serving/ingress/cmd/controller

FROM openshift/origin-base
USER 65532

COPY --from=builder /tmp/operator /ko-app/operator

ENTRYPOINT ["/ko-app/operator"]
