#!/usr/bin/env bash

function yq() {
  local thisfile rootdir bindir
  thisfile=$(realpath "${BASH_SOURCE[0]}")
  rootdir=$(dirname "$(dirname "$(dirname "${thisfile}")")")
  bindir="${rootdir}/_output/bin"
  # Prefer yq v3 from PATH (e.g. preinstalled in CI) over building it.
  if [[ -z "${YQ_BIN:-}" ]]; then
    if type -P yq > /dev/null && [[ "$(command yq --version 2>&1)" == *"version 3."* ]]; then
      YQ_BIN="$(type -P yq)"
    else
      if [[ ! -f "${bindir}/yq" ]]; then
        mkdir -p "${bindir}"
        GOBIN="${bindir}" GOFLAGS='' go install github.com/mikefarah/yq/v3@latest
      fi
      YQ_BIN="${bindir}/yq"
    fi
    export YQ_BIN
  fi
  "${YQ_BIN}" "$@"
}

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

  yq read "${metadata_file}" "${1}"
}
