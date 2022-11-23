#!/usr/bin/env bash

# Ensure -mod=vendor is removed
GOFLAGS="${GOFLAGS:-}"
export GOFLAGS="${GOFLAGS#-mod=vendor}"

#######################################
# Gets a value from a metadata file
# Globals:
#   None
# Arguments:
#   A metadata key path to get
# Outputs:
#   Writes metadata value on STDOUT
#######################################
function metadata.get {
  local metadata_file
  metadata_file="$(dirname "${BASH_SOURCE[0]}")/../../olm-catalog/serverless-operator/project.yaml"

  go run github.com/mikefarah/yq/v3@latest read "${metadata_file}" "${1}"
}
