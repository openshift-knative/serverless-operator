#!/bin/sh

cat /manifests/serverless-operator.clusterserviceversion.yaml | envsubst '$IMAGE_KNATIVE_OPERATOR $IMAGE_KNATIVE_OPENSHIFT_INGRESS' > /manifests/intermediate.yaml
mv /manifests/intermediate.yaml /manifests/serverless-operator.clusterserviceversion.yaml

cat /manifests/serverless-operator.clusterserviceversion.yaml