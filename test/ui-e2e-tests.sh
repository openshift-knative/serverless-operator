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
  logger.info "NodeJS version: $(node --version)"
  logger.info "NPM version: $(npm --version)"
}

function enable_dev_perspective() {
  local ocpversion
  ocpversion="$(oc get clusterversion/version -o jsonpath='{.status.desired.version}')"
  if versions.lt "$ocpversion" '4.19.0'; then
    logger.info 'Dev Console is always enabled for OCP <4.19. Skipping the enablement.'
    return
  fi
  local patch='{"spec":{"customization":{"perspectives":[{"id":"dev","visibility":{"state":"Enabled"}}]}}}'

  if LANG=C oc patch console.operator.openshift.io/cluster \
      --type='merge' \
      --dry-run='server' \
      -p "$patch" | grep -q 'no change'; then
    logger.success 'Dev Perspective already enabled'
    return
  fi

  oc patch console.operator.openshift.io/cluster \
    --type='merge' \
    -p "$patch"
  logger.success 'Dev Perspective enabled'
}

OCP_VERSION="$(oc get clusterversion version -o jsonpath='{.status.desired.version}')"
OCP_USERNAME="${OCP_USERNAME:-uitesting}"
OCP_PASSWORD="${OCP_PASSWORD:-$(echo "$OCP_USERNAME" | sha1sum - | awk '{print $1}')}"
OCP_LOGIN_PROVIDER="${OCP_LOGIN_PROVIDER:-my_htpasswd_provider}"
CYPRESS_BASE_URL="https://$(oc get route console -n openshift-console -o jsonpath='{.status.ingress[].host}')"

# Process arguments
DEFAULT_NPM_TARGET='test'
cypress_args=()
while [[ $# -gt 0 ]]; do
  case "$1" in
    --dev)
      DEFAULT_NPM_TARGET='dev'
      shift
      ;;
    --no-retries)
      cypress_args+=("--config" "retries=0")
      shift
      ;;
    *)
      # Pass through any other argument to cypress
      cypress_args+=("$1")
      shift
      ;;
  esac
done


if [ -n "${BUILD_ID:-}" ]; then
  export CYPRESS_NUM_TESTS_KEPT_IN_MEMORY=0
fi
export OCP_VERSION OCP_USERNAME OCP_PASSWORD OCP_LOGIN_PROVIDER CYPRESS_BASE_URL

add_user "$OCP_USERNAME" "$OCP_PASSWORD"

check_node
enable_dev_perspective
logger.success '🚀 Cluster prepared for testing.'

pushd "$(dirname "${BASH_SOURCE[0]}")/ui" >/dev/null
npm install
npm run install
npm run "${NPM_TARGET:-$DEFAULT_NPM_TARGET}" -- "${cypress_args[@]}"
