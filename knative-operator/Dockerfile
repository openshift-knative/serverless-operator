FROM openshift/origin-release:golang-1.13 AS builder

ENV SRC=${GOPATH}/src/github.com/openshift-knative/serverless-operator/knative-operator

WORKDIR ${SRC}
COPY . .

RUN go build -o /usr/bin/manager ./cmd/manager

FROM openshift/origin-base

ENV OPERATOR=/usr/local/bin/knative-serving-openshift \
    USER_UID=1001 \
    USER_NAME=knative-serving-openshift

COPY --from=builder /usr/bin/manager ${OPERATOR}

COPY build/bin /usr/local/bin
RUN /usr/local/bin/user_setup

# install manifest[s]
COPY deploy /deploy

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}