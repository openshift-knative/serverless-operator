FROM registry.ci.openshift.org/ocp/__OCP_MAX_VERSION__:cli-artifacts AS oc-image

FROM src

ARG TARGETARCH

COPY --from=oc-image /usr/share/openshift/linux_$TARGETARCH/oc.rhel8 /usr/bin/oc
COPY --from=registry.ci.openshift.org/openshift/knative-serving-src:__SERVING_VERSION__ /go/src/github.com/openshift-knative/serving/ /go/src/knative.dev/serving/
COPY --from=registry.ci.openshift.org/openshift/knative-eventing-src:__EVENTING_VERSION__ /go/src/github.com/openshift-knative/eventing/ /go/src/knative.dev/eventing/
COPY --from=registry.ci.openshift.org/openshift/eventing-kafka-broker-src:__EVENTING_KAFKA_BROKER_VERSION__ /go/src/github.com/openshift-knative/eventing-kafka-broker/ /go/src/knative.dev/eventing-kafka-broker/
COPY --from=registry.ci.openshift.org/openshift/eventing-istio-src:__EVENTING_ISTIO_VERSION__ /go/src/github.com/openshift-knative/eventing-istio/ /go/src/knative.dev/eventing-istio/

# Create a temp directory for the go_run() function that is writable by runtime users
ENV GORUN_PATH=/tmp/gorun
RUN mkdir -p /tmp/gorun && chmod g+rw /tmp/gorun

RUN chmod g+w /go/src/knative.dev/serving/ && chmod g+w /go/src/knative.dev/eventing/ && chmod g+w /go/src/knative.dev/eventing-kafka-broker/
