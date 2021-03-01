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

function check_node {
  if ! command -v npm >/dev/null 2>&1; then
    logger.error 'npm is required to run UI tests, install it.'
    return 51
  fi
}

function archive_cypress_artifacts {
  pushd "$(dirname "${BASH_SOURCE[0]}")/ui/cypress" >/dev/null
  mkdir -p "${ARTIFACTS}/ui/screenshots" "${ARTIFACTS}/ui/videos"
  ln -sf "${ARTIFACTS}/ui/screenshots" .
  ln -sf "${ARTIFACTS}/ui/videos" .
  popd >/dev/null
}

OCP_USERNAME="${OCP_USERNAME:-uitesting}"
OCP_PASSWORD="${OCP_PASSWORD:-$(echo "$OCP_USERNAME" | sha1sum - | awk '{print $1}')}"
OCP_LOGIN_PROVIDER="${OCP_LOGIN_PROVIDER:-my_htpasswd_provider}"
CYPRESS_BASE_URL="https://$(oc get route console -n openshift-console -o jsonpath='{.status.ingress[].host}')"
INSTALL_SERVERLESS="${INSTALL_SERVERLESS:-true}"
# use cypress:open to run test development UI
NPM_TARGET="${NPM_TARGET:-test}"
export OCP_USERNAME OCP_PASSWORD OCP_LOGIN_PROVIDER CYPRESS_BASE_URL

scale_up_workers
create_namespaces
add_user "$OCP_USERNAME" "$OCP_PASSWORD"
oc adm policy add-role-to-user edit "$OCP_USERNAME" -n "$TEST_NAMESPACE"
if [[ 'true' == "$INSTALL_SERVERLESS" ]]; then
  install_catalogsource
  ensure_serverless_installed
fi
check_node
archive_cypress_artifacts
logger.success '🚀 Cluster prepared for testing.'

pushd "$(dirname "${BASH_SOURCE[0]}")/ui" >/dev/null
npm install
npm run "${NPM_TARGET}"
