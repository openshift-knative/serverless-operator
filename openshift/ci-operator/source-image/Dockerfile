FROM src

COPY oc /usr/bin/oc
COPY --from=registry.ci.openshift.org/openshift/knative-v0.18.2:knative-serving-src /go/src/knative.dev/serving/ /go/src/knative.dev/serving/
COPY --from=registry.ci.openshift.org/openshift/knative-v0.18.6:knative-eventing-src /go/src/knative.dev/eventing/ /go/src/knative.dev/eventing/
COPY --from=registry.ci.openshift.org/openshift/knative-v0.18.0:knative-eventing-contrib-src /go/src/knative.dev/eventing-contrib/ /go/src/knative.dev/eventing-contrib/

RUN chmod g+w /go/src/knative.dev/serving/ && chmod g+w /go/src/knative.dev/eventing/ && chmod g+w /go/src/knative.dev/eventing-contrib/
