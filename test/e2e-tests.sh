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

# TODO: Split this to system namespaces and test namespaces
create_namespaces
# TODO: Move this to later stage (this is for tests)
create_htpasswd_users
add_roles

#install_catalogsource
logger.success '🚀 Cluster prepared for testing.'

# Run serverless-operator specific tests.
#serverless_operator_e2e_tests
if [[ $TEST_KNATIVE_KAFKA == true ]]; then
  #install_strimzi
  serverless_operator_kafka_e2e_tests
fi
#
#if [[ $FULL_MESH == "true" ]]; then
#  # net-istio does not use knative-serving-ingress namespace.
#  export INGRESS_NAMESPACE="knative-serving"
#  UNINSTALL_MESH="false" install_mesh
#  ensure_serverless_installed
#  enable_net_istio
#else
#  ensure_serverless_installed
#  trust_router_ca
#fi
#
#[ -n "$OPENSHIFT_CI" ] && setup_quick_api_deprecation_alerts
#
## Run Knative Serving & Eventing downstream E2E tests.
#downstream_serving_e2e_tests
#downstream_eventing_e2e_tests
#downstream_monitoring_e2e_tests
#if [[ $TEST_KNATIVE_KAFKA == true ]]; then
#  # TODO: Remove this as this is already installed in ensure_serverless_installed, or use deploy_knativekafka_cr
#  ensure_kafka_no_auth
#  downstream_knative_kafka_e2e_tests
#  # ensure_kafka_tls_auth
#  # downstream_knative_kafka_e2e_tests
#  # ensure_kafka_sasl_auth
#  # downstream_knative_kafka_e2e_tests
#fi
#
#[ -n "$OPENSHIFT_CI" ] && check_serverless_alerts
#
#teardown_serverless

success
