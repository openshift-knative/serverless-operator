#!/usr/bin/env bash

# For SC2164
set -e

function upstream_knative_eventing_e2e {
  should_run "${FUNCNAME[0]}" || return 0

  logger.info 'Running eventing tests'

  if [[ $FULL_MESH = true ]]; then
    upstream_knative_eventing_e2e_mesh
    return $?
  fi

  export TEST_IMAGE_TEMPLATE="registry.ci.openshift.org/openshift/knative-eventing-test-{{.Name}}:${KNATIVE_EVENTING_VERSION}"

  # shellcheck disable=SC1091
  source "${KNATIVE_EVENTING_HOME}/openshift/e2e-common.sh"

  cd "${KNATIVE_EVENTING_HOME}"

  # run_e2e_tests defined in knative-eventing
  logger.info 'Starting eventing e2e tests'
  run_e2e_tests

  # run_conformance_tests defined in knative-eventing
  logger.info 'Starting eventing conformance tests'
  run_conformance_tests
}

function upstream_knative_eventing_e2e_mesh() {
  pushd "${KNATIVE_EVENTING_ISTIO_HOME}" || return $?

  ./openshift/e2e-tests.sh || return $?

  popd || return $?
}
