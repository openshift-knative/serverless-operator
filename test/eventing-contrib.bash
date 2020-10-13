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

  # This the namespace used to install and test Knative Eventing-Contrib.
  export TEST_EVENTING_NAMESPACE
  TEST_EVENTING_NAMESPACE="${TEST_EVENTING_NAMESPACE:-"knative-eventing-"$(cat /dev/urandom \
    | tr -dc 'a-z0-9' | fold -w 10 | head -n 1)}"

  export KNATIVE_DEFAULT_NAMESPACE
  KNATIVE_DEFAULT_NAMESPACE="knative-eventing"

  # Config tracing config.
  export CONFIG_TRACING_CONFIG
  CONFIG_TRACING_CONFIG="test/config/config-tracing.yaml"

  install_tracing

  source "${KNATIVE_EVENTING_CONTRIB_HOME}/openshift/e2e-common.sh"

  failed=0

  logger.info 'Installing Strimzi'
  (( !failed )) && install_strimzi || failed=$?

  # run_e2e_tests defined in eventing-contrib
  logger.info 'Starting eventing-contrib tests'
  (( !failed )) && run_e2e_tests_workaround || failed=$?

  print_test_result ${failed}

  return $failed
  )
}

function run_e2e_tests_workaround(){

  oc get ns ${TEST_EVENTING_NAMESPACE} 2>/dev/null || TEST_EVENTING_NAMESPACE="knative-eventing"
  sed "s/namespace: ${KNATIVE_DEFAULT_NAMESPACE}/namespace: ${TEST_EVENTING_NAMESPACE}/g" ${CONFIG_TRACING_CONFIG} | oc replace -f -
  local test_name="${1:-}"
  local run_command=""
  local failed=0
  local channels=messaging.knative.dev/v1alpha1:KafkaChannel,messaging.knative.dev/v1beta1:KafkaChannel

  local common_opts=" -channels=$channels --kubeconfig $KUBECONFIG" ## --imagetemplate $TEST_IMAGE_TEMPLATE"
  if [ -n "$test_name" ]; then
      local run_command="-run ^(${test_name})$"
  fi

  go_test_e2e -timeout=90m -parallel=12 ./test/e2e \
    "$run_command" \
    $common_opts --dockerrepo "quay.io/openshift-knative" --tag "v0.17" || failed=$?

  return $failed
}
