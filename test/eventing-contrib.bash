#!/usr/bin/env bash

function upstream_knative_eventing_contrib_e2e {
  logger.info 'Running eventing-contrib tests'

  export TEST_IMAGE_TEMPLATE="registry.svc.ci.openshift.org/openshift/knative-${KNATIVE_EVENTING_CONTRIB_VERSION}:knative-eventing-test-{{.Name}}"

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

  logger.info 'Installing Tracing'
  install_tracing

  # shellcheck disable=SC1090
  source "${KNATIVE_EVENTING_CONTRIB_HOME}/openshift/e2e-common.sh"

  logger.info 'Installing Strimzi'
  install_strimzi

  # run_e2e_tests defined in eventing-contrib
  logger.info 'Starting eventing-contrib tests'
  run_e2e_tests
}
