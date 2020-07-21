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
scale_up_workers || exit $?
create_namespaces || exit $?
create_htpasswd_users && add_roles || exit $?

failed=0

(( !failed )) && install_catalogsource || failed=1
(( !failed )) && logger.success '🚀 Cluster prepared for testing.'

# TEST_ALL used in nightly CI where all tests are run in one go
if [[ $TEST_ALL == true ]]; then
  # Run upgrade tests
  (( !failed )) && install_serverless_previous || failed=2
  (( !failed )) && run_knative_serving_rolling_upgrade_tests || failed=3
  (( !failed )) && teardown_serverless || failed=4
  # Run serverless-operator specific tests
  (( !failed )) && serverless_operator_e2e_tests || failed=5
  # Run upstream knative serving & eventing tests
  (( !failed )) && ensure_serverless_installed || failed=6
  # Run knative serving additional e2e tests
  (( !failed )) && downstream_serving_e2e_tests || failed=7
else
  (( !failed )) && ensure_serverless_installed || failed=6
fi

# Run upstream knative serving & eventing tests
(( !failed )) && upstream_knative_serving_e2e_and_conformance_tests || failed=8
(( !failed )) && upstream_knative_eventing_e2e || failed=9

(( failed )) && dump_state
(( failed )) && exit $failed

success
