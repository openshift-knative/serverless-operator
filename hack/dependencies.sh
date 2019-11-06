#!/usr/bin/env bash

# This script can be used to install dependencies on a configured cluster. 
#
# This script will:
#  * Scale up cluster to accept serverless
#  * Install Red Hat Service Mesh and it's dependencies
#  * Adds namespace fo Knative Serving
#  * Configure Service Mesh Member Roll
#

# shellcheck disable=SC1091,SC1090
source "$(dirname "${BASH_SOURCE[0]}")/lib/__sources__.bash"

set -Eeuo pipefail

scale_up_workers || exit $?
create_namespaces || exit $?

exitcode=0

(( !exitcode )) && install_service_mesh_operator || exitcode=2

exit $exitcode
