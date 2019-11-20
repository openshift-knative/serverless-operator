FROM registry.access.redhat.com/ubi7/ubi-minimal:latest

ENV OPERATOR=/usr/local/bin/knative-serving-openshift \
    USER_UID=1001 \
    USER_NAME=knative-serving-openshift

# install operator binary
COPY build/_output/bin/knative-serving-openshift ${OPERATOR}

COPY build/bin /usr/local/bin
RUN  /usr/local/bin/user_setup

# install manifest[s]
COPY deploy /deploy

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}
