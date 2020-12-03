#!/usr/bin/env bash

function upstream_knative_eventing_contrib_e2e {
  (
  logger.info 'Running eventing-contrib tests'

  local failed=0
  local randomns

  TEST_IMAGE_TEMPLATE="registry.svc.ci.openshift.org/openshift/knative-${KNATIVE_EVENTING_CONTRIB_VERSION}:knative-eventing-test-{{.Name}}"

  cd "$KNATIVE_EVENTING_CONTRIB_HOME" || return $?

  # This the namespace used to install and test Knative Eventing-Contrib.
  randomns="knative-eventing-$(LC_ALL=C dd if=/dev/urandom bs=256 count=1 2> /dev/null \
    | tr -dc 'a-z0-9' | fold -w 10 | head -n 1)"
  TEST_EVENTING_NAMESPACE="${TEST_EVENTING_NAMESPACE:-"${randomns}"}"
  export TEST_EVENTING_NAMESPACE

  export KNATIVE_DEFAULT_NAMESPACE
  KNATIVE_DEFAULT_NAMESPACE="knative-eventing"

  # Config tracing config.
  export CONFIG_TRACING_CONFIG
  CONFIG_TRACING_CONFIG="test/config/config-tracing.yaml"

  install_tracing

  source "${KNATIVE_EVENTING_CONTRIB_HOME}/openshift/e2e-common.sh"

  failed=0

  # run_e2e_tests defined in eventing-contrib
  logger.info 'Starting eventing-contrib tests'
  (( !failed )) && create_auth_secrets || failed=$?

  (( !failed )) && run_e2e_tests || failed=$?

  return $failed
  )
}
