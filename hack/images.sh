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

# Dockerfiles might include references to images that do not exit but CI operator
# will automatically replace them with proper images during CI builds (as long as
# the string starts with registry.ci.openshift.org). For non-CI builds,
# some images need to be replaced with upstream variants manually.
function replace_images() {
  dockerfile_path=${1:?Pass dockerfile path}
  tmp_dockerfile=$(mktemp /tmp/Dockerfile.XXXXXX)
  sed -e "s|registry.ci.openshift.org/ocp/\(.*\):base|quay.io/openshift/origin-base:\1|" \
    "${dockerfile_path}" > "$tmp_dockerfile"
  echo "$tmp_dockerfile"
}

function build_image() {
  name=${1:?Pass a name of image to be built as arg[1]}
  dockerfile_path=${2:?Pass dockerfile path}
  tmp_dockerfile=$(replace_images "${dockerfile_path}")

  logger.info "Using ${tmp_dockerfile} as Dockerfile"

  if ! oc get buildconfigs "$name" -n "$OLM_NAMESPACE" >/dev/null 2>&1; then
    logger.info "Create an image build for ${name}"
    oc -n "${OLM_NAMESPACE}" new-build \
      --strategy=docker --name "$name" --dockerfile "$(cat "${tmp_dockerfile}")"
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
  tmp_dockerfile=$(replace_images openshift-knative-operator/Dockerfile)
  podman build -t "$repo/openshift-knative-operator" -f "${tmp_dockerfile}" .
  podman push "$repo/openshift-knative-operator"

  tmp_dockerfile=$(replace_images knative-operator/Dockerfile)
  podman build -t "$repo/knative-operator" -f "${tmp_dockerfile}" .
  podman push "$repo/knative-operator"

  tmp_dockerfile=$(replace_images serving/ingress/Dockerfile)
  podman build -t "$repo/knative-openshift-ingress" -f "${tmp_dockerfile}" .
  podman push "$repo/knative-openshift-ingress"

fi
