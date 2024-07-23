FROM registry.access.redhat.com/ubi8-minimal:latest AS builder

RUN microdnf install -y golang

COPY . /code
WORKDIR /code

ENV GOFLAGS="-mod=vendor"
RUN go build -o /tmp/metadata-webhook github.com/openshift-knative/serverless-operator/serving/metadata-webhook/cmd/webhook

FROM registry.access.redhat.com/ubi8-minimal:latest
USER 65532

COPY --from=builder /tmp/metadata-webhook /ko-app/metadata-webhook

ENTRYPOINT ["/ko-app/metadata-webhook"]
