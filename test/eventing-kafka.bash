#!/usr/bin/env bash

# For SC2164
set -e

function upstream_knative_eventing_kafka_broker_e2e {
  should_run "${FUNCNAME[0]}" || return 0

  if [[ $FULL_MESH = true ]]; then
    # upstream_knative_eventing_e2e_mesh function in eventing.bash runs:
    # - Eventing core tests
    # - EKB tests
    # in the mesh case, so we don't need to do anything here
    return 0
  fi

  logger.info 'Setting Kafka as default broker class'

  local root_dir
  root_dir="$(dirname "$(dirname "$(realpath "${BASH_SOURCE[0]}")")")"

  # Set Kafka Broker as default Broker class
  oc patch knativeeventing --type merge -n knative-eventing knative-eventing --patch-file "${root_dir}/test/config/eventing/kafka-broker-default-patch.yaml"

  logger.info 'Running eventing-kafka-broker tests'

  cd "$KNATIVE_EVENTING_KAFKA_BROKER_HOME"

  export FIRST_EVENT_DELAY_ENABLED=false # Disable very slow test since it's already running in ekb CI

  export TEST_IMAGE_TEMPLATE="registry.ci.openshift.org/openshift/knative-eventing-kafka-broker-test-{{.Name}}:${KNATIVE_EVENTING_KAFKA_BROKER_VERSION}"

  export SKIP_GENERATE_RELEASE=true

  # || true -> Suppress errors due to readonly or undefined variables
  # shellcheck disable=SC1091,SC1090
  source "${KNATIVE_EVENTING_KAFKA_BROKER_HOME}/openshift/e2e-common.sh" || true

  logger.info 'Starting eventing-kafka-broker tests'

  export SYSTEM_NAMESPACE="${EVENTING_NAMESPACE}"

  run_e2e_tests
  run_conformance_tests
  run_e2e_new_tests

  # Rollback setting Kafka as default Broker class
  oc patch knativeeventing --type merge -n knative-eventing knative-eventing --patch-file "${root_dir}/test/config/eventing/kafka-broker-default-patch-rollback.yaml"
}
