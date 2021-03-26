#!/usr/bin/env bash

# For SC2164
set -e

readonly EVENTING_READY_FILE="/tmp/eventing-prober-ready"
readonly EVENTING_PROBER_FILE="/tmp/eventing-prober-signal"

function upstream_knative_eventing_e2e {
  logger.info 'Running eventing tests'

  export TEST_IMAGE_TEMPLATE="registry.ci.openshift.org/openshift/knative-${KNATIVE_EVENTING_VERSION}:knative-eventing-test-{{.Name}}"

  cd "${KNATIVE_EVENTING_HOME}"

  # shellcheck disable=SC1090
  source "${KNATIVE_EVENTING_HOME}/openshift/e2e-common.sh"

  # run_e2e_tests defined in knative-eventing
  run_e2e_tests
}

function prepare_knative_eventing_tests {
  logger.info 'Nothing to prepare for Eventing upgrade tests'
}
