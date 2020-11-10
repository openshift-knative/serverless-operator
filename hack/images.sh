#!/usr/bin/env bash

# This script can be used to publish all images built by
# this repository to the specified docker repository.

set -e

if [[ "$#" -ne 1 ]]; then
    echo "Please ensure DOCKER_REPO_OVERRIDE envvar is set"
    exit 1
fi

repo=$1

docker build -t "$repo/openshift-knative-operator" -f openshift-knative-operator/Dockerfile .
docker push "$repo/openshift-knative-operator"

docker build -t "$repo/knative-operator" -f knative-operator/Dockerfile .
docker push "$repo/knative-operator"
