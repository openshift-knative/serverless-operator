FROM src

COPY oc /usr/bin/oc
COPY --from=registry.svc.ci.openshift.org/openshift/knative-v0.16.0:knative-serving-src /go/src/knative.dev/serving/ /go/src/knative.dev/serving/
COPY --from=registry.svc.ci.openshift.org/openshift/knative-v0.16.0:knative-eventing-src /go/src/knative.dev/eventing/ /go/src/knative.dev/eventing/
