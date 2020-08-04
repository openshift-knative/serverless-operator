#!/bin/sh

if [ -n "$BUILD" ]; then
  ns=$(echo $BUILD | jq -r .metadata.namespace)
  base="registry.svc.ci.openshift.org/$ns/stable:"
  export IMAGE_KNATIVE_OPERATOR="${base}knative-operator"
  export IMAGE_KNATIVE_OPENSHIFT_INGRESS="${base}knative-openshift-ingress"
elif [ -n "$DOCKER_REPO_OVERRIDE" ]; then
  export IMAGE_KNATIVE_OPERATOR="${DOCKER_REPO_OVERRIDE}/knative-operator"
  export IMAGE_KNATIVE_OPENSHIFT_INGRESS="${DOCKER_REPO_OVERRIDE}/knative-openshift-ingress"
else
  export IMAGE_KNATIVE_OPERATOR="registry.svc.ci.openshift.org/openshift/openshift-serverless-1.8.0:knative-operator"
  export IMAGE_KNATIVE_OPENSHIFT_INGRESS="registry.svc.ci.openshift.org/openshift/openshift-serverless-1.8.0:knative-openshift-ingress"
fi

cat /manifests/serverless-operator.clusterserviceversion.yaml | envsubst '$IMAGE_KNATIVE_OPERATOR $IMAGE_KNATIVE_OPENSHIFT_INGRESS' > /manifests/intermediate.yaml
mv /manifests/intermediate.yaml /manifests/serverless-operator.clusterserviceversion.yaml

cat /manifests/serverless-operator.clusterserviceversion.yaml