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

teardown_serverless
create_namespaces "${SYSTEM_NAMESPACES[@]}"
install_catalogsource
logger.success '🚀 Cluster prepared for testing.'

create_namespaces "${TEST_NAMESPACES[@]}"
# Install ServiceMesh and enable mTLS.
if [[ $FULL_MESH == true ]]; then
  UNINSTALL_MESH="false" install_mesh
else
  trust_router_ca
fi

# Run upgrade tests
if [[ $TEST_KNATIVE_UPGRADE == true ]]; then
  install_serverless_previous
  # Set KafkaChannel as default for upgrade tests.
  if [[ $INSTALL_KAFKA == "true" ]]; then
    ensure_kafka_channel_default
  fi
  run_rolling_upgrade_tests
  teardown_serverless
fi

# Run upstream knative serving, eventing and eventing-kafka tests
if [[ $TEST_KNATIVE_E2E == true ]]; then
  ensure_serverless_installed
  if [[ $TEST_KNATIVE_KAFKA == true ]]; then
    upstream_knative_eventing_kafka_e2e
  fi
  upstream_knative_serving_e2e_and_conformance_tests
  upstream_knative_eventing_e2e
fi

[ -n "$OPENSHIFT_CI" ] && check_serverless_alerts

success
