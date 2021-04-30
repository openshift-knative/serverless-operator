#!/usr/bin/env bash

function upstream_knative_eventing_e2e {
  (
  logger.info 'Running eventing tests'

  local failed=0

  TEST_IMAGE_TEMPLATE="registry.ci.openshift.org/openshift/knative-${KNATIVE_EVENTING_VERSION}:knative-eventing-test-{{.Name}}"

  cd "$KNATIVE_EVENTING_HOME" || return $?

  source "${KNATIVE_EVENTING_HOME}/openshift/e2e-common.sh"

  # run_e2e_tests defined in knative-eventing
  run_e2e_tests || failed=$?

  print_test_result ${failed}

  return $failed
  )
}
