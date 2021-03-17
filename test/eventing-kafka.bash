#!/usr/bin/env bash

# For SC2164
set -e

function upstream_knative_eventing_kafka_e2e {
  logger.info 'Running eventing-kafka tests'

  local randomns

  export TEST_IMAGE_TEMPLATE="registry.ci.openshift.org/openshift/knative-${KNATIVE_EVENTING_KAFKA_VERSION}:knative-eventing-test-{{.Name}}"

  cd "$KNATIVE_EVENTING_KAFKA_HOME"

  # This the namespace used to install and test Knative Eventing-Kafka.
  randomns="knative-eventing-$(LC_ALL=C dd if=/dev/urandom bs=256 count=1 2> /dev/null \
    | tr -dc 'a-z0-9' | fold -w 10 | head -n 1)"
  SYSTEM_NAMESPACE="${SYSTEM_NAMESPACE:-"${randomns}"}"
  export SYSTEM_NAMESPACE
  KNATIVE_DEFAULT_NAMESPACE=$SYSTEM_NAMESPACE
  export KNATIVE_DEFAULT_NAMESPACE

  # Config tracing config.
  export CONFIG_TRACING_CONFIG
  CONFIG_TRACING_CONFIG="test/config/config-tracing.yaml"

  # shellcheck disable=SC1090
  source "${KNATIVE_EVENTING_KAFKA_HOME}/openshift/e2e-common.sh"

  logger.info 'Installing Tracing'
  install_tracing

  # run_e2e_tests defined in eventing-kafka
  logger.info 'Starting eventing-kafka tests'
  run_e2e_tests
}
