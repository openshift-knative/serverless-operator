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
dump_state.setup

use_spot_instances
scale_up_workers
create_namespaces "${SYSTEM_NAMESPACES[@]}"

if [[ $INSTALL_CERTMANAGER == "true" ]]; then
  install_certmanager
fi
ensure_content_source_policy
ensure_catalogsource_installed
ensure_serverless_installed
