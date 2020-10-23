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

create_namespaces
teardown_serverless
install_catalogsource
logger.success '🚀 Cluster prepared for testing.'

# Run upgrade tests
if [[ $TEST_KNATIVE_UPGRADE == true ]]; then
  # TODO(markusthoemmes): Remove after 1.11 is cut.
  oc create namespace "${SERVING_NAMESPACE}"

  install_serverless_previous
  run_knative_serving_rolling_upgrade_tests
  trigger_gc_and_print_knative
  # Call teardown only if E2E tests follow.
  if [[ $TEST_KNATIVE_E2E == true ]]; then
    teardown_serverless
  fi
fi

# Run upstream knative serving, eventing and eventing-contrib tests
if [[ $TEST_KNATIVE_E2E == true ]]; then
  # Need 6 worker nodes when running upstream.
  SCALE_UP="${SCALE_UP:-6}" scale_up_workers
  ensure_serverless_installed
  if [[ $TEST_KNATIVE_KAFKA == true ]]; then
    upstream_knative_eventing_contrib_e2e
  fi
  upstream_knative_serving_e2e_and_conformance_tests
  upstream_knative_eventing_e2e
fi

success
