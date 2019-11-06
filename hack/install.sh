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

scale_up_workers || exit $?
create_namespaces || exit $?

exitcode=0

(( !exitcode )) && install_service_mesh_operator || exitcode=2
(( !exitcode )) && ensure_catalogsource_installed || exitcode=3
(( !exitcode )) && ensure_serverless_installed || exitcode=4

exit $exitcode
