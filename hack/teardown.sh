#!/usr/bin/env bash

# This script can be used to teardown Serverless on a configured cluster. 
#
# This script will:
#  
#  * Remove Knative Serving custom resource
#  * Uninstall Serverless Operator
#  * Remove Catalog Source
#  * Unregister namespaces from Service Mesh and remove them
#

# shellcheck disable=SC1091,SC1090
source "$(dirname "${BASH_SOURCE[0]}")/lib/__sources__.bash"

set -Eeuo pipefail

exitcode=0

(( !exitcode )) && teardown_serverless || exitcode=2
(( !exitcode )) && delete_catalog_source || exitcode=3
(( !exitcode )) && delete_namespaces || exitcode=4

exit $exitcode
