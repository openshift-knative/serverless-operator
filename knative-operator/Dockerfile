FROM registry.ci.openshift.org/openshift/release:rhel-8-release-golang-1.22-openshift-4.16 AS builder

ENV BASE=github.com/openshift-knative/serverless-operator
WORKDIR ${GOPATH}/src/${BASE}

COPY . .

ENV GOFLAGS="-mod=vendor"
RUN go build -o /tmp/operator ${BASE}/knative-operator/cmd/manager

FROM registry.access.redhat.com/ubi8-minimal:latest
USER 65532

COPY --from=builder /tmp/operator /ko-app/operator
# install manifest[s]
COPY knative-operator/deploy /deploy

ENTRYPOINT ["/ko-app/operator"]
