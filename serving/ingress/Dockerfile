FROM openshift/origin-release:golang-1.14 AS builder

ENV SRC=${GOPATH}/src/github.com/openshift-knative/serverless-operator/serving/ingress

WORKDIR ${SRC}
COPY . .

RUN go build -o /usr/bin/controller ./cmd/controller

FROM openshift/origin-base

ENV OPERATOR=/usr/local/bin/knative-openshift-ingress \
    USER_UID=1001 \
    USER_NAME=knative-openshift-ingress

COPY --from=builder /usr/bin/controller ${OPERATOR}

COPY build/bin /usr/local/bin
RUN /usr/local/bin/user_setup

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}