#!/usr/bin/env bash

# Make sure yq is on PATH.
yq > /dev/null || exit 127

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
  local matadata_file
  matadata_file="$(dirname "${BASH_SOURCE[0]}")/../../metadata.yaml"

  yq read "${matadata_file}" "${1}"
}
