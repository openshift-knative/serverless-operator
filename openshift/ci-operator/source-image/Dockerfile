FROM src

COPY --from=registry.ci.openshift.org/ocp/4.5:cli /usr/bin/oc /usr/bin/oc
COPY --from=registry.ci.openshift.org/openshift/knative-v0.16.0:knative-serving-src /go/src/knative.dev/serving/ /go/src/knative.dev/serving/
COPY --from=registry.ci.openshift.org/openshift/knative-v0.16.0:knative-eventing-src /go/src/knative.dev/eventing/ /go/src/knative.dev/eventing/
