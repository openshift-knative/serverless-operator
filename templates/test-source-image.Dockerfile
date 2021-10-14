FROM src

COPY oc /usr/bin/oc
COPY --from=quay.io/openshift-knative/knative-serving-src:v__SERVING_VERSION__ /go/src/knative.dev/serving/ /go/src/knative.dev/serving/
COPY --from=quay.io/openshift-knative/knative-eventing-src:v__EVENTING_VERSION__ /go/src/knative.dev/eventing/ /go/src/knative.dev/eventing/
COPY --from=quay.io/openshift-knative/knative-eventing-kafka-src:v__EVENTING_KAFKA_VERSION__ /go/src/knative.dev/eventing-kafka/ /go/src/knative.dev/eventing-kafka/

RUN chmod g+w /go/src/knative.dev/serving/ && chmod g+w /go/src/knative.dev/eventing/ && chmod g+w /go/src/knative.dev/eventing-kafka/
