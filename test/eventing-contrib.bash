#!/usr/bin/env bash

# TO BE REMOVED
function header_text {
  header $1
}
function upstream_knative_eventing_contrib_e2e {
  (
  logger.info 'Running eventing-contrib tests'

  local failed=0

  TEST_IMAGE_TEMPLATE="registry.svc.ci.openshift.org/openshift/knative-${KNATIVE_EVENTING_CONTRIB_VERSION}:knative-eventing-test-{{.Name}}"

  cd "$KNATIVE_EVENTING_CONTRIB_HOME" || return $?

  source "${KNATIVE_EVENTING_CONTRIB_HOME}/vendor/knative.dev/eventing/test/e2e-common.sh"
  source "${KNATIVE_EVENTING_CONTRIB_HOME}/openshift/e2e-common.sh"

  failed=0

  logger.info 'Installing Strimzi'
  (( !failed )) && install_strimzi || failed=$?

  # run_e2e_tests defined in eventing-contrib
  logger.info 'Starting eventing-contrib tests'
  (( !failed )) && run_e2e_tests || failed=$?

  print_test_result ${failed}

  return $failed
  )
}
