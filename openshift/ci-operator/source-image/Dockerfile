FROM src

COPY oc /usr/bin/oc
COPY --from=registry.ci.openshift.org/openshift/knative-v0.23.1:knative-serving-src /go/src/knative.dev/serving/ /go/src/knative.dev/serving/
COPY --from=registry.ci.openshift.org/openshift/knative-v0.23.0:knative-eventing-src /go/src/knative.dev/eventing/ /go/src/knative.dev/eventing/
COPY --from=registry.ci.openshift.org/openshift/knative-v0.23.3:knative-eventing-kafka-src /go/src/knative.dev/eventing-kafka/ /go/src/knative.dev/eventing-kafka/

RUN chmod g+w /go/src/knative.dev/serving/ && chmod g+w /go/src/knative.dev/eventing/ && chmod g+w /go/src/knative.dev/eventing-kafka/
