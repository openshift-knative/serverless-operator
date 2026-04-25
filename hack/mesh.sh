#!/usr/bin/env bash

# This script can be used to install ServiceMesh on a configured cluster. 
#
# This script will:
#  * Install ServiceMesh
#

# shellcheck disable=SC1091,SC1090
source "$(dirname "${BASH_SOURCE[0]}")/lib/__sources__.bash"

set -Eeuo pipefail

debugging.setup

if [[ ${UNINSTALL_MESH:-} == "true" ]]; then
  if [[ ${MESH_VERSION:-2} == "3" ]]; then
    uninstall_mesh3
  else
    uninstall_mesh
  fi
else
  if [[ ${MESH_VERSION:-2} == "3" ]]; then
    install_mesh3
  else
    install_mesh
  fi
fi
