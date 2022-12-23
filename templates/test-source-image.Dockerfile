FROM src

COPY oc /usr/bin/oc
COPY --from=registry.ci.openshift.org/openshift/knative-v__SERVING_VERSION__:knative-serving-src /go/src/knative.dev/serving/ /go/src/knative.dev/serving/
COPY --from=registry.ci.openshift.org/openshift/knative-eventing-src:__EVENTING_VERSION__ /go/src/github.com/openshift-knative/eventing/ /go/src/knative.dev/eventing/
COPY --from=registry.ci.openshift.org/openshift/knative-v__EVENTING_KAFKA_VERSION__:knative-eventing-kafka-src /go/src/knative.dev/eventing-kafka/ /go/src/knative.dev/eventing-kafka/
COPY --from=registry.ci.openshift.org/openshift/eventing-kafka-broker-src:__EVENTING_KAFKA_BROKER_VERSION__ /go/src/github.com/openshift-knative/eventing-kafka-broker/ /go/src/knative.dev/eventing-kafka-broker/

# Create a temp directory for the go_run() function that is writable by runtime users
ENV GORUN_PATH=/tmp/gorun
RUN mkdir /tmp/gorun && chmod g+rw /tmp/gorun

RUN chmod g+w /go/src/knative.dev/serving/ && chmod g+w /go/src/knative.dev/eventing/ && chmod g+w /go/src/knative.dev/eventing-kafka/ && chmod g+w /go/src/knative.dev/eventing-kafka-broker/
