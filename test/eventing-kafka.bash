#!/usr/bin/env bash

# For SC2164
set -e

function upstream_knative_eventing_kafka_e2e {
  logger.info 'Running eventing-kafka tests'

  local random_ns

  export TEST_IMAGE_TEMPLATE="registry.ci.openshift.org/openshift/knative-${KNATIVE_EVENTING_KAFKA_VERSION}:knative-eventing-kafka-test-{{.Name}}"

  cd "$KNATIVE_EVENTING_KAFKA_HOME"

  # This the namespace used to install and test Knative Eventing-Kafka.
  random_ns="knative-eventing-$(LC_ALL=C dd if=/dev/urandom bs=256 count=1 2>/dev/null |
    tr -dc 'a-z0-9' | fold -w 10 | head -n 1)"
  SYSTEM_NAMESPACE="${SYSTEM_NAMESPACE:-"${random_ns}"}"
  export SYSTEM_NAMESPACE

  # Config tracing config.
  export CONFIG_TRACING_CONFIG
  CONFIG_TRACING_CONFIG="test/config/config-tracing.yaml"

  # shellcheck disable=SC1091
  source "${KNATIVE_EVENTING_KAFKA_HOME}/openshift/e2e-common.sh"

  # run_e2e_tests defined in eventing-kafka
  logger.info 'Starting eventing-kafka tests'
  run_e2e_tests
}

function upstream_knative_eventing_kafka_broker_e2e {
  logger.info 'Setting Kafka as default broker class'

  local root_dir
  root_dir="$(dirname "$(dirname "$(realpath "${BASH_SOURCE[0]}")")")"

  # Set Kafka Broker as default Broker class
  oc patch knativeeventing --type merge -n knative-eventing knative-eventing --patch-file "${root_dir}/test/config/eventing/kafka-broker-default-patch.yaml"

  logger.info 'Running eventing-kafka-broker tests'

  cd "$KNATIVE_EVENTING_KAFKA_BROKER_HOME"

  export FIRST_EVENT_DELAY_ENABLED=false # Disable very slow test since it's already running in ekb CI

  # Mock image env variables for TEST_IMAGE_TEMPLATE
  export KNATIVE_EVENTING_KAFKA_BROKER_TEST_EVENT_SENDER=""
  export KNATIVE_EVENTING_KAFKA_BROKER_TEST_HEARTBEATS=""
  export KNATIVE_EVENTING_KAFKA_BROKER_TEST_EVENTSHUB=""
  export KNATIVE_EVENTING_KAFKA_BROKER_TEST_RECORDEVENTS=""
  export KNATIVE_EVENTING_KAFKA_BROKER_TEST_PRINT=""
  export KNATIVE_EVENTING_KAFKA_BROKER_TEST_PERFORMANCE=""
  export KNATIVE_EVENTING_KAFKA_BROKER_TEST_EVENT_FLAKER=""
  export KNATIVE_EVENTING_KAFKA_BROKER_TEST_EVENT_LIBRARY=""
  export KNATIVE_EVENTING_KAFKA_BROKER_TEST_COMMITTED_OFFSET=""
  export KNATIVE_EVENTING_KAFKA_BROKER_TEST_CONSUMER_GROUP_LAG_PROVIDER_TEST=""
  export KNATIVE_EVENTING_KAFKA_BROKER_TEST_KAFKA_CONSUMER=""
  export KNATIVE_EVENTING_KAFKA_BROKER_TEST_PARTITIONS_REPLICATION_VERIFIER=""
  export KNATIVE_EVENTING_KAFKA_BROKER_TEST_REQUEST_SENDER=""

  # shellcheck disable=SC1091,SC1090
  source "${KNATIVE_EVENTING_KAFKA_BROKER_HOME}/openshift/e2e-common.sh"

  logger.info 'Starting eventing-kafka-broker tests'

  export TEST_IMAGE_TEMPLATE="registry.ci.openshift.org/openshift/knative-eventing-kafka-broker-test-{{.Name}}:${KNATIVE_EVENTING_KAFKA_BROKER_VERSION}"
  export SYSTEM_NAMESPACE="${EVENTING_NAMESPACE}"

  run_e2e_tests
  run_conformance_tests
  run_e2e_new_tests

  # Rollback setting Kafka as default Broker class
  oc patch knativeeventing --type merge -n knative-eventing knative-eventing --patch-file "${root_dir}/test/config/eventing/kafka-broker-default-patch-rollback.yaml"
}
