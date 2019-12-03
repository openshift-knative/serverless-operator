#!/usr/bin/env bash

# This script can be used to publish all images built by
# this repository to the specified docker repository.

repo=$1

(
cd serving/operator || exit 1
docker build -t "$repo/knative-serving-operator" .
docker push "$repo/knative-serving-operator"
)

(
cd serving/ingress || exit 1
docker build -t "$repo/knative-openshift-ingress" .
docker push "$repo/knative-openshift-ingress"
)