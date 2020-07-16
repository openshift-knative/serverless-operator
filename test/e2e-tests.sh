#!/usr/bin/env bash

# shellcheck disable=SC1091,SC1090
source "$(dirname "${BASH_SOURCE[0]}")/lib.bash"

set -Eeuo pipefail

# Enable extra verbosity if running in CI.
if [ -n "$OPENSHIFT_CI" ]; then
  env
fi
debugging.setup

register_teardown || exit $?
create_namespaces || exit $?
create_htpasswd_users && add_roles || exit $?

failed=0

(( !failed )) && install_catalogsource || failed=1
(( !failed )) && logger.success '🚀 Cluster prepared for testing.'

if [[ ${TEST_ALL:=false} == true ]]; then
  (( !failed )) && install_serverless_previous || failed=2
  (( !failed )) && run_knative_serving_rolling_upgrade_tests || failed=3
  (( !failed )) && teardown_serverless || failed=4
fi

# Run serverless-operator specific tests.
(( !failed )) && serverless_operator_e2e_tests || failed=5

# Run upstream knative serving & eventing tests
(( !failed )) && ensure_serverless_installed || failed=6

# Run knative serving additional e2e tests
(( !failed )) && downstream_serving_e2e_tests || failed=7

if [[ $TEST_ALL == true ]]; then
  (( !failed )) && upstream_knative_serving_e2e_and_conformance_tests || failed=8
  (( !failed )) && knative_eventing_tests || failed=9
fi

(( failed )) && dump_state
(( failed )) && exit $failed

success
