FROM src

COPY oc /usr/bin/oc
<<<<<<< HEAD
COPY --from=registry.ci.openshift.org/openshift/knative-v0.26.0:knative-serving-src /go/src/knative.dev/serving/ /go/src/knative.dev/serving/
=======
<<<<<<< HEAD
COPY --from=registry.ci.openshift.org/openshift/knative-v0.25.1:knative-serving-src /go/src/knative.dev/serving/ /go/src/knative.dev/serving/
>>>>>>> 4dff7c67 (:recycle: :dizzy: :whale: Moving a few images from OCP registry to quay.io, and regenerating the actual manifests and related files)
COPY --from=registry.ci.openshift.org/openshift/knative-v0.26.0:knative-eventing-src /go/src/knative.dev/eventing/ /go/src/knative.dev/eventing/
COPY --from=registry.ci.openshift.org/openshift/knative-v0.25.3:knative-eventing-kafka-src /go/src/knative.dev/eventing-kafka/ /go/src/knative.dev/eventing-kafka/
=======
COPY --from=quay.io/openshift-knative/knative-serving-src:v0.25.1 /go/src/knative.dev/serving/ /go/src/knative.dev/serving/
COPY --from=quay.io/openshift-knative/knative-eventing-src:v0.25.1 /go/src/knative.dev/eventing/ /go/src/knative.dev/eventing/
COPY --from=quay.io/openshift-knative/knative-eventing-kafka-src:v0.25.3 /go/src/knative.dev/eventing-kafka/ /go/src/knative.dev/eventing-kafka/
>>>>>>> 388bdd16 (:recycle: :dizzy: :whale: Moving a few images from OCP registry to quay.io, and regenerating the actual manifests and related files)

RUN chmod g+w /go/src/knative.dev/serving/ && chmod g+w /go/src/knative.dev/eventing/ && chmod g+w /go/src/knative.dev/eventing-kafka/
