FROM registry.ci.openshift.org/origin/scos-4.20:cli-artifacts AS oc-image

FROM src

ARG TARGETARCH

COPY --from=oc-image /usr/share/openshift/linux_$TARGETARCH/oc.rhel8 /usr/bin/oc
COPY --from=registry.ci.openshift.org/openshift/knative-serving-src:knative-v1.17 /go/src/github.com/openshift-knative/serving/ /go/src/knative.dev/serving/
COPY --from=registry.ci.openshift.org/openshift/knative-eventing-src:knative-v1.17 /go/src/github.com/openshift-knative/eventing/ /go/src/knative.dev/eventing/
COPY --from=registry.ci.openshift.org/openshift/eventing-kafka-broker-src:knative-v1.17 /go/src/github.com/openshift-knative/eventing-kafka-broker/ /go/src/knative.dev/eventing-kafka-broker/
COPY --from=registry.ci.openshift.org/openshift/eventing-istio-src:knative-v1.17 /go/src/github.com/openshift-knative/eventing-istio/ /go/src/knative.dev/eventing-istio/

# Create a temp directory for the go_run() function that is writable by runtime users
ENV GORUN_PATH=/tmp/gorun
RUN mkdir -p /tmp/gorun && chmod g+rw /tmp/gorun

RUN chmod g+w /go/src/knative.dev/serving/ && chmod g+w /go/src/knative.dev/eventing/ && chmod g+w /go/src/knative.dev/eventing-kafka-broker/
