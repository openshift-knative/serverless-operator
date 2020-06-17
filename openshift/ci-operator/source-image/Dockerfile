FROM src

COPY oc /usr/bin/oc
COPY serving ${GOPATH}/src/knative.dev/serving
COPY --from=registry.svc.ci.openshift.org/openshift/knative-v0.14.2:knative-eventing-src /go/src/knative.dev/eventing/ /go/src/knative.dev/eventing/