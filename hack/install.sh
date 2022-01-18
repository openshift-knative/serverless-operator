#!/usr/bin/env bash

# This script can be used to install Serverless on a configured cluster. 
#
# This script will:
#  * Install and configure dependencies
#  * Install Serverless Operator from this repository
#  * Install Knative Serving custom resource
#

# shellcheck disable=SC1091,SC1090
source "$(dirname "${BASH_SOURCE[0]}")/lib/__sources__.bash"

set -Eeuo pipefail

debugging.setup
if [[ "$DUMP_STATE" == "true" ]]; then
  dump_state.setup
fi

create_namespaces "${SYSTEM_NAMESPACES[@]}"

ensure_catalogsource_installed
ensure_serverless_installed "${INSTALL_PREVIOUS_VERSION}"
