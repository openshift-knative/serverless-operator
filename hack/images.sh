#!/usr/bin/env bash

# This script can be used to publish all images built by
# this repository to the specified docker repository.

set -Eeuo pipefail

# shellcheck disable=SC1091,SC1090
source "$(dirname "${BASH_SOURCE[0]}")/lib/__sources__.bash"

root_dir="$(dirname "$(dirname "$(realpath "${BASH_SOURCE[0]}")")")"

if [[ "$#" -ne 1 ]]; then
  echo "Please ensure DOCKER_REPO_OVERRIDE envvar is set"
  exit 1
fi

repo=$1

on_cluster_builds=${ON_CLUSTER_BUILDS:-false}
echo "On cluster builds: ${on_cluster_builds}"

if [[ $on_cluster_builds = true ]]; then
  #  image-registry.openshift-image-registry.svc:5000/openshift-marketplace/openshift-knative-operator:latest
  build_image "serverless-openshift-knative-operator" "${root_dir}" "openshift-knative-operator/Dockerfile" || exit 1
  #  image-registry.openshift-image-registry.svc:5000/openshift-marketplace/knative-operator:latest
  build_image "serverless-knative-operator" "${root_dir}" "knative-operator/Dockerfile" || exit 1
  #  image-registry.openshift-image-registry.svc:5000/openshift-marketplace/knative-openshift-ingress:latest
  build_image "serverless-ingress" "${root_dir}" "serving/ingress/Dockerfile" || exit 1

  logger.info 'Images build'

else
  CONTAINER_BUILD_TAG_SUFFIX=${CONTAINER_BUILD_TAG_SUFFIX:-""}
  tmp_dockerfile=$(replace_images openshift-knative-operator/Dockerfile)
  podman build -t "$repo/serverless-openshift-knative-operator${CONTAINER_BUILD_TAG_SUFFIX}"  -f "${tmp_dockerfile}" .
  podman push "$repo/serverless-openshift-knative-operator${CONTAINER_BUILD_TAG_SUFFIX}"

  tmp_dockerfile=$(replace_images knative-operator/Dockerfile)
  podman build -t "$repo/serverless-knative-operator${CONTAINER_BUILD_TAG_SUFFIX}" -f "${tmp_dockerfile}" .
  podman push "$repo/serverless-knative-operator${CONTAINER_BUILD_TAG_SUFFIX}"

  tmp_dockerfile=$(replace_images serving/ingress/Dockerfile)
  podman build -t "$repo/serverless-ingress${CONTAINER_BUILD_TAG_SUFFIX}" -f "${tmp_dockerfile}" .
  podman push "$repo/serverless-ingress${CONTAINER_BUILD_TAG_SUFFIX}"
fi
