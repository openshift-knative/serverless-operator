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

scale_up_workers
create_namespaces "${SYSTEM_NAMESPACES[@]}"

export ON_CLUSTER_BUILDS=true
export DOCKER_REPO_OVERRIDE="image-registry.openshift-image-registry.svc:5000/openshift-marketplace"

make images

ensure_catalogsource_installed
ensure_serverless_installed "${INSTALL_PREVIOUS_VERSION}"
