FROM src

COPY oc /usr/bin/oc
COPY --from=registry.svc.ci.openshift.org/openshift/knative-v0.17.3:knative-serving-src /go/src/knative.dev/serving/ /go/src/knative.dev/serving/
COPY --from=registry.svc.ci.openshift.org/openshift/knative-v0.17.2:knative-eventing-src /go/src/knative.dev/eventing/ /go/src/knative.dev/eventing/
COPY --from=registry.svc.ci.openshift.org/openshift/knative-v0.17.1:knative-eventing-contrib-src /go/src/knative.dev/eventing-contrib/ /go/src/knative.dev/eventing-contrib/

RUN chmod g+w /go/src/knative.dev/serving/ && chmod g+w /go/src/knative.dev/eventing/ && chmod g+w /go/src/knative.dev/eventing-contrib/
