#!/usr/bin/env bash

# For SC2164
set -e

function upstream_knative_eventing_e2e {
  should_run "upstream_knative_eventing_e2e" || return

  logger.info 'Running eventing tests'

  export TEST_IMAGE_TEMPLATE="registry.ci.openshift.org/openshift/knative-eventing-test-{{.Name}}:${KNATIVE_EVENTING_VERSION}"

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
