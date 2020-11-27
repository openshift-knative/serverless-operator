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

scale_up_workers
create_namespaces
create_htpasswd_users
add_roles

install_catalogsource
logger.success '🚀 Cluster prepared for testing.'

# Run serverless-operator specific tests.
serverless_operator_e2e_tests
if [[ $TEST_KNATIVE_KAFKA == true ]]; then
  install_strimzi
  serverless_operator_kafka_e2e_tests
fi
ensure_serverless_installed
# Run Knative Serving & Eventing downstream E2E tests.
downstream_serving_e2e_tests
downstream_eventing_e2e_tests
if [[ $TEST_KNATIVE_KAFKA == true ]]; then
  ensure_kafka_no_auth
  downstream_knative_kafka_e2e_tests
  ensure_kafka_tls_auth
  downstream_knative_kafka_e2e_tests
  ensure_kafka_sasl_auth
  downstream_knative_kafka_e2e_tests
fi

success
