#!/usr/bin/env bash

# For SC2164
set -e

function upstream_knative_eventing_e2e {
  logger.info 'Running eventing tests'

  export TEST_IMAGE_TEMPLATE="registry.ci.openshift.org/openshift/knative-${KNATIVE_EVENTING_VERSION}:knative-eventing-test-{{.Name}}"

  cd "${KNATIVE_EVENTING_HOME}"

  # shellcheck disable=SC1090
  source "${KNATIVE_EVENTING_HOME}/openshift/e2e-common.sh"

  logger.info 'Installing Tracing'
  install_tracing

  # run_e2e_tests defined in knative-eventing
  logger.info 'Starting eventing e2e tests'
  run_e2e_tests

  # run_conformance_tests defined in knative-eventing
  logger.info 'Starting eventing conformance tests'
  run_conformance_tests
}

function prepare_knative_eventing_tests {
  logger.info 'Nothing to prepare for Eventing upgrade tests'
}
