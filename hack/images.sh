#!/usr/bin/env bash

# This script can be used to publish all images built by
# this repository to the specified docker repository.

set -e

repo=$1

docker build -t "$repo/knative-operator" knative-operator
docker push "$repo/knative-operator"

docker build -t "$repo/knative-openshift-ingress" serving/ingress
docker push "$repo/knative-openshift-ingress"

docker build --build-arg repo=$repo -t "$repo/openshift-serverless-bundle" olm-catalog/serverless-operator
docker push "$repo/openshift-serverless-bundle"
