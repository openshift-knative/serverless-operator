#!/usr/bin/env bash

function checkout_knative_eventing_operator {
  checkout_repo 'knative.dev/eventing-operator' \
    "${KNATIVE_EVENTING_OPERATOR_REPO}" \
    "${KNATIVE_EVENTING_OPERATOR_VERSION}" \
    "${KNATIVE_EVENTING_OPERATOR_BRANCH}"
}

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

function knative_eventing_operator_tests {
  logger.info 'Running eventing operator tests'
  (
  local exitstatus=0

  checkout_knative_eventing_operator

  export TEST_NAMESPACE="${EVENTING_NAMESPACE}"

  go_test_e2e -timeout=20m -parallel=1 ./test/e2e \
    --kubeconfig "$KUBECONFIG" \
    || exitstatus=$? && true

  print_test_result ${exitstatus}

  remove_temporary_gopath

  return $exitstatus
  )
}
