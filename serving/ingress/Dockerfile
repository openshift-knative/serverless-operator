FROM openshift/origin-release:golang-1.14 AS builder
COPY . .

WORKDIR serving/ingress
ENV GOFLAGS="-mod=vendor"
RUN go build -o /tmp/operator ./cmd/controller

FROM openshift/origin-base
COPY --from=builder /tmp/operator /ko-app/operator

ENTRYPOINT ["/ko-app/operator"] 
