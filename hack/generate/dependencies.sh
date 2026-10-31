#!/usr/bin/env bash

set -Eeuo pipefail

# Upgrades the pinned dependency images in project.yaml (service-mesh-proxy,
# kube-rbac-proxy, func utils). These end up in the CSV, so run `make
# generated-files` afterwards to regenerate it with the updated images.
# Requires registry credentials (REGISTRY_REDHAT_IO_USERNAME/PASSWORD) for
# registry.redhat.io, which is why this is kept separate from generated-files.

# shellcheck disable=SC1091,SC1090
source "$(dirname "${BASH_SOURCE[0]}")/../lib/__sources__.bash"

function upgrade_service_mesh_proxy_image() {
  sm_proxy_image=$(yq r olm-catalog/serverless-operator/project.yaml 'dependencies.service_mesh_proxy')
  # shellcheck disable=SC2086
  sm_proxy_image_stream=$(skopeo inspect --retry-times=10 --no-tags=true "docker://${sm_proxy_image}" ${SKOPEO_EXTRA_FLAGS} | jq -r '.Labels.version')
  sm_proxy_image_stream=${sm_proxy_image_stream%.*}
  sm_proxy_image=$(latest_konflux_image_sha "${sm_proxy_image}" "${sm_proxy_image_stream}")
  yq w --inplace olm-catalog/serverless-operator/project.yaml 'dependencies.service_mesh_proxy' "${sm_proxy_image}"
}

function upgrade_kube_rbac_proxy_image() {
  local image image_stream
  image=$(metadata.get 'dependencies.kube_rbac_proxy')
  image_stream=$(metadata.get 'requirements.ocpVersion.kube-rbac-proxy')
  image=$(latest_konflux_image_sha "${image}" "v${image_stream}")
  yq w --inplace olm-catalog/serverless-operator/project.yaml 'dependencies.kube_rbac_proxy' "${image}"
}

function upgrade_func_images() {
  # List of images in project.yaml dependencies.func
  local images=("tekton_buildah" "nodejs_20_minimal" "openjdk_21" "python-39")
  for img_name in "${images[@]}" ; do
    logger.info "Upgrading func image: ${img_name}"
    local image image_stream
    image=$(metadata.get "dependencies.func.${img_name}")
    image_stream="latest"
    image=$(latest_konflux_image_sha "${image}" "${image_stream}")
    yq w --inplace olm-catalog/serverless-operator/project.yaml "dependencies.func.${img_name}" "${image}"
  done
}

function upgrade_dependencies_images {
  if [[ -n "${REGISTRY_REDHAT_IO_USERNAME:-}" ]] || [[ -n "${REGISTRY_REDHAT_IO_PASSWORD:-}" ]]; then
    skopeo login registry.redhat.io -u "${REGISTRY_REDHAT_IO_USERNAME}" -p "${REGISTRY_REDHAT_IO_PASSWORD}"
  fi

  upgrade_service_mesh_proxy_image

  upgrade_kube_rbac_proxy_image

  upgrade_func_images
}

logger.info "Upgrading registry.redhat.io images"
upgrade_dependencies_images
