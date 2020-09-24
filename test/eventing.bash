#!/usr/bin/env bash

function upstream_knative_eventing_e2e {
  (
  logger.info 'Running eventing tests'

  local failed=0

  TEST_IMAGE_TEMPLATE="registry.svc.ci.openshift.org/openshift/knative-${KNATIVE_EVENTING_VERSION}:knative-eventing-test-{{.Name}}"

  logger.info 'File system permissions'

  whoami

  groups

  pwd

  ls -al

  cd "$KNATIVE_EVENTING_HOME" || return $?

  pwd

  ls -al

  > testfile

  source "${KNATIVE_EVENTING_HOME}/openshift/e2e-common.sh"

  cd test

  ls -al

  > testfile

  cd -

  # run_e2e_tests defined in knative-eventing
  #run_e2e_tests || failed=$?

  #print_test_result ${failed}

  return $failed
  )
}
