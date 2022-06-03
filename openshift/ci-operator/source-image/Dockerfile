FROM src

COPY oc /usr/bin/oc
COPY --from=registry.ci.openshift.org/openshift/knative-v1.3.0:knative-serving-src /go/src/knative.dev/serving/ /go/src/knative.dev/serving/
COPY --from=registry.ci.openshift.org/openshift/knative-v1.2.1:knative-eventing-src /go/src/knative.dev/eventing/ /go/src/knative.dev/eventing/
COPY --from=registry.ci.openshift.org/openshift/knative-v1.1.0:knative-eventing-kafka-src /go/src/knative.dev/eventing-kafka/ /go/src/knative.dev/eventing-kafka/
COPY --from=registry.ci.openshift.org/openshift/knative-v1.2.3:knative-eventing-kafka-broker-src /go/src/knative.dev/eventing-kafka-broker/ /go/src/knative.dev/eventing-kafka-broker/

RUN chmod g+w /go/src/knative.dev/serving/ && chmod g+w /go/src/knative.dev/eventing/ && chmod g+w /go/src/knative.dev/eventing-kafka/ && chmod g+w /go/src/knative.dev/eventing-kafka-broker/
