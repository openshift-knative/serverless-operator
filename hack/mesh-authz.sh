#!/usr/bin/env bash

# Apply or remove mesh AuthorizationPolicies.
# Called separately from mesh.sh to allow applying policies AFTER
# KnativeServing is ready, avoiding a race condition where the
# activator→autoscaler websocket gets blocked by deny-all-by-default
# before ALLOW policies are loaded by the sidecars.

# shellcheck disable=SC1091,SC1090
source "$(dirname "${BASH_SOURCE[0]}")/lib/__sources__.bash"

set -Eeuo pipefail

debugging.setup

if [[ ${UNINSTALL_MESH:-} == "true" ]]; then
  undeploy_mesh3_authorization_policies
else
  deploy_mesh3_authorization_policies
fi
