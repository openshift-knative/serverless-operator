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

if [[ $UNINSTALL_MESH == "true" ]]; then
  uninstall_mesh
else
  install_mesh
fi
