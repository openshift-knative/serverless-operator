FROM openshift/origin-release:golang-1.12 AS builder

ENV SRC=${GOPATH}/src/github.com/openshift-knative/serverless-operator/serving/ingress
WORKDIR ${SRC}

COPY . .

RUN go build ./cmd/manager
RUN mv manager /usr/bin/manager

FROM openshift/origin-base

ENV OPERATOR=/usr/local/bin/knative-openshift-ingress \
    USER_UID=1001 \
    USER_NAME=knative-openshift-ingress

COPY --from=builder /usr/bin/manager ${OPERATOR}

COPY build/bin /usr/local/bin
RUN  /usr/local/bin/user_setup

# install manifest[s]
COPY deploy /deploy

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}