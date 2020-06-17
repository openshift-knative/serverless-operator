FROM src

COPY oc /usr/bin/oc
COPY serving ${GOPATH}/src/knative.dev/serving
COPY --from=registry.svc.ci.openshift.org/openshift/knative-v0.14.2:knative-eventing-src ${GOPATH}/src/knative.dev/eventing/ ${GOPATH}/src/knative.dev/eventing/