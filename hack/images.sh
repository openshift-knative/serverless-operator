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

function build_image() {
  name=${1:?Pass a name of image to be built as arg[1]}
  dockerfile_path=${2:?Pass dockerfile path}

  if ! oc get buildconfigs "$name" -n "$OLM_NAMESPACE" >/dev/null 2>&1; then
    logger.info "Create an image build for ${name}"
    oc -n "${OLM_NAMESPACE}" new-build \
      --strategy=docker --name "$name" --dockerfile "$(cat "${dockerfile_path}")"
  else
    logger.info "${name} image build is already created"
  fi

  logger.info 'Build the image in the cluster-internal registry.'
  oc -n "${OLM_NAMESPACE}" start-build "${name}" --from-dir "${root_dir}" -F
}

if [[ $on_cluster_builds = true ]]; then
  #  image-registry.openshift-image-registry.svc:5000/openshift-marketplace/openshift-knative-operator:latest
  build_image "openshift-knative-operator" "${root_dir}/openshift-knative-operator/Dockerfile" || exit 1
  #  image-registry.openshift-image-registry.svc:5000/openshift-marketplace/knative-operator:latest
  build_image "knative-operator" "${root_dir}/knative-operator/Dockerfile" || exit 1
  #  image-registry.openshift-image-registry.svc:5000/openshift-marketplace/knative-openshift-ingress:latest
  build_image "knative-openshift-ingress" "${root_dir}/serving/ingress/Dockerfile" || exit 1

  logger.info 'Images build'

else
  docker build -t "$repo/openshift-knative-operator" -f openshift-knative-operator/Dockerfile .
  docker push "$repo/openshift-knative-operator"

  docker build -t "$repo/knative-operator" -f knative-operator/Dockerfile .
  docker push "$repo/knative-operator"

  docker build -t "$repo/knative-openshift-ingress" -f serving/ingress/Dockerfile .
  docker push "$repo/knative-openshift-ingress"

fi
