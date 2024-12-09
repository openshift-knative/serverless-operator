#!/usr/bin/env bash

# This script can be used to install the Serverless operator on a cluster.
# It's to be used a development script and doesn't scale the cluster or
# anything like that.
#
# This script will:
#  * Install and configure dependencies
#  * Install Serverless Operator from this repository
#

# shellcheck disable=SC1091,SC1090
source "$(dirname "${BASH_SOURCE[0]}")/lib/__sources__.bash"

set -Eeuo pipefail

debugging.setup
if [[ ${INSTALL_WITH_ARGO_CD:-} != "true" ]]; then
  create_namespaces "${SYSTEM_NAMESPACES[@]}"
fi
ensure_catalogsource_installed
