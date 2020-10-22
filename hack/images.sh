#!/usr/bin/env bash

# This script can be used to publish all images built by
# this repository to the specified docker repository.

set -Eeuo pipefail

if [[ "$#" -ne 1 ]]; then
    echo "Please ensure DOCKER_REPO_OVERRIDE envvar is set"
    exit 1
fi

repo=$1

docker build -t "$repo/openshift-knative-operator" openshift-knative-operator
docker push "$repo/openshift-knative-operator"

docker build -t "$repo/knative-operator" knative-operator
docker push "$repo/knative-operator"

docker build -t "$repo/knative-openshift-ingress" serving/ingress
docker push "$repo/knative-openshift-ingress"

