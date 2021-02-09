#!/usr/bin/env bash

# shellcheck disable=SC1091,SC1090
source "$(dirname "${BASH_SOURCE[0]}")/lib.bash"

set -Eeuo pipefail

# Enable extra verbosity if running in CI.
if [ -n "$OPENSHIFT_CI" ]; then
  env
fi
debugging.setup
dump_state.setup

function check_node() {
  if ! command -v npm >/dev/null 2>&1; then
    logger.error 'npm is required to run UI tests, install it.'
    return 51
  fi
}

scale_up_workers
create_namespaces
create_htpasswd_users
ensure_catalogsource_installed
ensure_serverless_installed
check_node
logger.success '🚀 Cluster prepared for testing.'

pushd "$(dirname "${BASH_SOURCE[0]}")/ui"
npm install

env OCP_LOGIN_PROVIDER="${OCP_LOGIN_PROVIDER:-my_htpasswd_provider}" \
  OCP_USERNAME="${OCP_USERNAME:-user1}" \
  OCP_PASSWORD="${OCP_PASSWORD:-password1}" \
  CYPRESS_BASE_URL="https://$(oc get route console -n openshift-console \
  -o jsonpath='{.status.ingress[].host}')" \
  npm run test -- headed
