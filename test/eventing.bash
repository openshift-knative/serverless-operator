#!/usr/bin/env bash

# For SC2164
set -e

function upstream_knative_eventing_e2e {
  logger.info 'Running eventing tests'

  export TEST_IMAGE_TEMPLATE=${IMAGE_REGISTRY_NAME}/openshift-knative-eventing-test/{{.Name}}:v1.3

  cd "${KNATIVE_EVENTING_HOME}"

  # shellcheck disable=SC1091
  source "${KNATIVE_EVENTING_HOME}/openshift/e2e-common.sh"

  # run_e2e_tests defined in knative-eventing
  logger.info 'Starting eventing e2e tests'
  run_e2e_tests

  # run_conformance_tests defined in knative-eventing
  logger.info 'Starting eventing conformance tests'
  run_conformance_tests
}
