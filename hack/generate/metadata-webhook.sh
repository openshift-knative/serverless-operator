#!/usr/bin/env bash

set -Eeuo pipefail

# shellcheck disable=SC1091,SC1090
source "$(dirname "${BASH_SOURCE[0]}")/../lib/__sources__.bash"

function update_webhook_image {
  local image="${registry_quay}/serverless-metadata-webhook:latest"
  yq w --inplace serving/metadata-webhook/config/webhook.yaml 'spec.template.spec.containers(name==webhook).image' "${image}"
}

logger.info "Updating metadata-webhook image"
update_webhook_image
