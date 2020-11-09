FROM openshift/origin-release:golang-1.14 AS builder
COPY . .
ENV GOFLAGS="-mod=vendor"
RUN go build -o /tmp/operator ./cmd/manager

FROM openshift/origin-base
COPY --from=builder /tmp/operator /ko-app/operator
# install manifest[s]
COPY deploy /deploy

ENTRYPOINT ["/ko-app/operator"]
