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

create_namespaces "${SYSTEM_NAMESPACES[@]}"
install_catalogsource

if [[ $INSTALL_KAFKA == true ]]; then
  install_strimzi
fi

if [[ $FULL_MESH == "true" ]]; then
  # net-istio does not use knative-serving-ingress namespace.
  export INGRESS_NAMESPACE="knative-serving"
  UNINSTALL_MESH="false" install_mesh
  ensure_serverless_installed
  enable_net_istio
else
  ensure_serverless_installed
  trust_router_ca
fi

logger.success '🚀 Cluster prepared for testing.'

# Run serverless-operator specific tests.
create_htpasswd_users && add_roles
create_namespaces "${TEST_NAMESPACES[@]}"
serverless_operator_e2e_tests
if [[ $TEST_KNATIVE_KAFKA == true ]]; then
  serverless_operator_kafka_e2e_tests
fi

[ -n "$OPENSHIFT_CI" ] && setup_quick_api_deprecation_alerts

# Run Knative Serving & Eventing downstream E2E tests.
downstream_serving_e2e_tests
downstream_eventing_e2e_tests
downstream_monitoring_e2e_tests
if [[ $TEST_KNATIVE_KAFKA == true ]]; then
  # TODO: Remove this as this is already installed in ensure_serverless_installed, or use deploy_knativekafka_cr
  ensure_kafka_no_auth
  downstream_knative_kafka_e2e_tests
  # ensure_kafka_tls_auth
  # downstream_knative_kafka_e2e_tests
  # ensure_kafka_sasl_auth
  # downstream_knative_kafka_e2e_tests
fi

[ -n "$OPENSHIFT_CI" ] && check_serverless_alerts

teardown_serverless

success
