#!/usr/bin/env bash

function knative_eventing_tests {
  (
  local exitstatus=0
  logger.info 'Running eventing tests'

  cd "$KNATIVE_EVENTING_HOME" || return $?

  image_template="registry.svc.ci.openshift.org/openshift/knative-${KNATIVE_EVENTING_VERSION}:knative-eventing-test-{{.Name}}"

  go_test_e2e -timeout=90m -parallel=1 ./test/e2e \
    --kubeconfig "$KUBECONFIG" \
    --imagetemplate "$image_template" \
    || exitstatus=$? && true

  print_test_result ${exitstatus}

  return $exitstatus
  )
}
