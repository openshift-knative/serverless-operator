#!/usr/bin/env bash

function checkout_knative_eventing {
  checkout_repo 'knative.dev/eventing' \
    "${KNATIVE_EVENTING_REPO}" \
    "${KNATIVE_EVENTING_VERSION}" \
    "${KNATIVE_EVENTING_BRANCH}"
}

function checkout_knative_eventing_operator {
  checkout_repo 'knative.dev/eventing-operator' \
    "${KNATIVE_EVENTING_OPERATOR_REPO}" \
    "${KNATIVE_EVENTING_OPERATOR_VERSION}" \
    "${KNATIVE_EVENTING_OPERATOR_BRANCH}"
}

function run_knative_eventing_tests {
  (
  local exitstatus=0
  logger.info 'Running eventing tests'

  checkout_knative_eventing

  go_test_e2e -timeout=90m -parallel=1 ./test/e2e \
    --kubeconfig "$KUBECONFIG" \
    --dockerrepo 'quay.io/openshift-knative' \
    || exitstatus=$? && true

  if (( !exitstatus )); then
    logger.success 'Tests have passed'
  else
    logger.error 'Tests have failures!'
  fi

  remove_temporary_gopath

  return $exitstatus
  )
}

function run_knative_eventing_operator_tests {
  (
  local exitstatus=0
  logger.info 'Running eventing operator tests'

  checkout_knative_eventing_operator

  logger.warn 'TODO(ksuszyns): Write eventing operator tests execution.'

  if (( !exitstatus )); then
    logger.success 'Tests have passed'
  else
    logger.error 'Tests have failures!'
  fi

  remove_temporary_gopath

  return $exitstatus
  )
}
