#!/usr/bin/env bash

function upstream_knative_eventing_contrib_e2e {
  (
  logger.info 'Running eventing-contrib tests'

  local failed=0

  TEST_IMAGE_TEMPLATE="registry.svc.ci.openshift.org/openshift/knative-${KNATIVE_EVENTING_CONTRIB_VERSION}:knative-eventing-contrib-test-{{.Name}}"

  cd "$KNATIVE_EVENTING_CONTRIB_HOME" || return $?

  source "${KNATIVE_EVENTING_CONTRIB_HOME}/openshift/e2e-common.sh"

  # run_e2e_tests defined in eventing-contrib
  run_e2e_tests || failed=$?

  print_test_result ${failed}

  return $failed
  )
}
