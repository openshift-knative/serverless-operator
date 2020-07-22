#!/usr/bin/env bash

# shellcheck disable=SC1091,SC1090
source "$(dirname "${BASH_SOURCE[0]}")/lib.bash"

set -Eeuo pipefail

# Enable extra verbosity if running in CI.
if [ -n "$OPENSHIFT_CI" ]; then
  env
fi
debugging.setup

scale_up_workers || exit $?
create_namespaces || exit $?

failed=0

(( !failed )) && teardown_serverless || failed=1
(( !failed )) && install_catalogsource || failed=2
(( !failed )) && logger.success '🚀 Cluster prepared for testing.'

# Run upgrade tests
if [[ $TEST_KNATIVE_UPGRADE == true ]]; then
  (( !failed )) && install_serverless_previous || failed=3
  (( !failed )) && run_knative_serving_rolling_upgrade_tests || failed=4
  (( !failed )) && trigger_gc_and_print_knative || failed=5
  (( !failed )) && teardown_serverless || failed=6
fi

# Run upstream knative serving & eventing tests
if [[ $TEST_KNATIVE_E2E == true ]]; then
  (( !failed )) && ensure_serverless_installed || failed=7
  (( !failed )) && upstream_knative_serving_e2e_and_conformance_tests || failed=8
  (( !failed )) && upstream_knative_eventing_e2e || failed=9
fi

(( failed )) && dump_state
(( failed )) && exit $failed

success
