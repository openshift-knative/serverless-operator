#!/bin/sh

export IMAGE_KNATIVE_OPERATOR="registry.svc.ci.openshift.org/openshift/openshift-serverless-v1.7.2:knative-operator"
export IMAGE_KNATIVE_OPENSHIFT_INGRESS="registry.svc.ci.openshift.org/openshift/openshift-serverless-1.7.2:knative-openshift-ingress"


cat /manifests/serverless-operator.clusterserviceversion.yaml | envsubst '$IMAGE_KNATIVE_OPERATOR $IMAGE_KNATIVE_OPENSHIFT_INGRESS' > /manifests/intermediate.yaml
mv /manifests/intermediate.yaml /manifests/serverless-operator.clusterserviceversion.yaml

cat /manifests/serverless-operator.clusterserviceversion.yaml