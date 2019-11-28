FROM registry.access.redhat.com/ubi7/ubi-minimal:7.6

ENV OPERATOR=/usr/local/bin/knative-serving-operator \
    USER_UID=1001 \
    USER_NAME=knative-serving-operator

# install operator binary
COPY build/_output/bin/knative-serving-operator ${OPERATOR}

COPY build/bin /usr/local/bin
RUN  /usr/local/bin/user_setup

# install manifest[s]
COPY deploy /deploy

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}
