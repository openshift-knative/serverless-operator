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
install_catalogsource
ensure_serverless_installed
check_node
export OCP_USERNAME="${OCP_USERNAME:-user1}"
oc apply -f - <<EOF
kind: RoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: ${OCP_USERNAME}-edit
  namespace: ${TEST_NAMESPACE}
subjects:
  - kind: User
    apiGroup: rbac.authorization.k8s.io
    name: ${OCP_USERNAME}
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: edit
EOF
logger.success '🚀 Cluster prepared for testing.'

pushd "$(dirname "${BASH_SOURCE[0]}")/ui"
npm install

env OCP_LOGIN_PROVIDER="${OCP_LOGIN_PROVIDER:-my_htpasswd_provider}" \
  OCP_PASSWORD="${OCP_PASSWORD:-password1}" \
  CYPRESS_BASE_URL="https://$(oc get route console -n openshift-console \
  -o jsonpath='{.status.ingress[].host}')" \
  npm run test
