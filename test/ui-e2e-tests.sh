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
  mkdir -p "${ARTIFACTS}/ui/screenshots" "${ARTIFACTS}/ui/videos" "${ARTIFACTS}/ui/results"
  pushd "$(dirname "${BASH_SOURCE[0]}")/ui/cypress" >/dev/null
  ln -sf "${ARTIFACTS}/ui/screenshots" "${ARTIFACTS}/ui/videos" .
  popd >/dev/null
  pushd "$(dirname "${BASH_SOURCE[0]}")/ui" >/dev/null
  ln -sf "${ARTIFACTS}/ui/results" .
  popd >/dev/null
}

function ui_tests {
  should_run "${FUNCNAME[0]}" || return 0

  OCP_VERSION="$(oc get clusterversion version -o jsonpath='{.status.desired.version}')"
  OCP_USERNAME="${OCP_USERNAME:-uitesting}"
  OCP_PASSWORD="${OCP_PASSWORD:-$(echo "$OCP_USERNAME" | sha1sum - | awk '{print $1}')}"
  OCP_LOGIN_PROVIDER="${OCP_LOGIN_PROVIDER:-my_htpasswd_provider}"
  CYPRESS_BASE_URL="https://$(oc get route console -n openshift-console -o jsonpath='{.status.ingress[].host}')"
  # use dev to run test development UI
  NPM_TARGET="${NPM_TARGET:-test}"
  if [ -n "${BUILD_ID:-}" ]; then
    export CYPRESS_NUM_TESTS_KEPT_IN_MEMORY=0
  fi
  export OCP_VERSION OCP_USERNAME OCP_PASSWORD OCP_LOGIN_PROVIDER CYPRESS_BASE_URL

  create_namespaces "${TEST_NAMESPACES[@]}"
  add_user "$OCP_USERNAME" "$OCP_PASSWORD"
  oc adm policy add-role-to-user edit "$OCP_USERNAME" -n "$TEST_NAMESPACE"
  check_node
  archive_cypress_artifacts
  logger.success '🚀 Cluster prepared for testing.'

  pushd "$(dirname "${BASH_SOURCE[0]}")/ui" >/dev/null
  npm install
  npm run install
  npm run "${NPM_TARGET}"
}

ui_tests

