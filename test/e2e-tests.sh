#!/usr/bin/env bash

# shellcheck disable=SC1091,SC1090
source "$(dirname "${BASH_SOURCE[0]}")/lib.bash"

set -Eeuo pipefail

# Enable extra verbosity if running in CI.
if [ -n "$OPENSHIFT_CI" ]; then
  env
fi
debugging.setup # both install and test
dump_state.setup # test

if [[ $FULL_MESH == "true" ]]; then
  # net-istio does not use knative-serving-ingress namespace.
  export INGRESS_NAMESPACE="knative-serving"
else
  trust_router_ca
fi

logger.success '🚀 Cluster prepared for testing.'

# Run serverless-operator specific tests.
create_namespaces "${TEST_NAMESPACES[@]}"
link_global_pullsecret_to_namespaces "${TEST_NAMESPACES[@]}"
create_htpasswd_users && add_roles

run_testselect

serverless_operator_e2e_tests
if [[ $TEST_KNATIVE_KAFKA == true ]]; then
  serverless_operator_kafka_e2e_tests
fi

[ -n "$OPENSHIFT_CI" ] && setup_quick_api_deprecation_alerts

# Run Knative Serving & Eventing downstream E2E tests.
downstream_serving_e2e_tests
downstream_eventing_e2e_tests
downstream_eventing_e2e_rekt_tests
downstream_monitoring_e2e_tests
if [[ $TEST_KNATIVE_KAFKA == true ]]; then
  downstream_knative_kafka_e2e_tests
fi

[ -n "$OPENSHIFT_CI" ] && check_serverless_alerts

success
