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

logger.success '🚀 Cluster prepared for testing.'

create_namespaces "${TEST_NAMESPACES[@]}"

if [[ $MESH != true ]]; then
  trust_router_ca
fi

run_testselect

# Run upgrade tests
if [[ $TEST_KNATIVE_UPGRADE == true ]]; then
#  # Set KafkaChannel as default for upgrade tests.
#  if [[ $TEST_KNATIVE_KAFKA == "true" ]]; then
#    ensure_kafka_channel_default
#  fi
  run_rolling_upgrade_tests
fi

# Run upstream knative serving, eventing and eventing-kafka-broker tests
if [[ $TEST_KNATIVE_E2E == true ]]; then
  # TODO: Remove this when upstream tests can use in-cluster config.
  # See https://github.com/knative/eventing/issues/5996 (the same issue affects Eventing Kafka)
  ensure_kubeconfig
#  if [[ $TEST_KNATIVE_KAFKA_BROKER == true ]]; then
#    upstream_knative_eventing_kafka_broker_e2e
#  fi
  if [[ $TEST_KNATIVE_SERVING == true ]]; then
    upstream_knative_serving_e2e_and_conformance_tests
  fi
#  if [[ $TEST_KNATIVE_EVENTING == true ]]; then
#    upstream_knative_eventing_e2e
#  fi
fi

[ -n "$OPENSHIFT_CI" ] && check_serverless_alerts

success
