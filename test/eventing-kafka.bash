#!/usr/bin/env bash

# For SC2164
set -e

function upstream_knative_eventing_kafka_e2e {
  logger.info 'Running eventing-kafka tests'

  local random_ns

  export TEST_IMAGE_TEMPLATE="registry.ci.openshift.org/openshift/knative-${KNATIVE_EVENTING_KAFKA_BROKER_VERSION}:knative-eventing-kafka-broker-test-{{.Name}}"

  cd "$KNATIVE_EVENTING_KAFKA_BROKER_HOME"

  # This the namespace used to install and test Knative Eventing-Kafka.
  random_ns="knative-eventing-$(LC_ALL=C dd if=/dev/urandom bs=256 count=1 2> /dev/null \
    | tr -dc 'a-z0-9' | fold -w 10 | head -n 1)"
  SYSTEM_NAMESPACE="${SYSTEM_NAMESPACE:-"${random_ns}"}"
  export SYSTEM_NAMESPACE

  # Config tracing config.
  export CONFIG_TRACING_CONFIG
  CONFIG_TRACING_CONFIG="test/config/100-config-tracing.yaml"

  # shellcheck disable=SC1091
  source "${KNATIVE_EVENTING_KAFKA_BROKER_HOME}/openshift/e2e-common.sh"

  # run_e2e_tests defined in eventing-kafka
  logger.info 'Starting eventing-kafka tests'
  run_e2e_tests
}
