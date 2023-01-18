#!/usr/bin/env bash

# For SC2164
set -e

function upstream_knative_eventing_e2e {
  should_run "${FUNCNAME[0]}" || return

  logger.info 'Running eventing tests'

  export TEST_IMAGE_TEMPLATE="registry.ci.openshift.org/openshift/knative-eventing-test-{{.Name}}:${KNATIVE_EVENTING_VERSION}"

  # shellcheck disable=SC1091
  source "${KNATIVE_EVENTING_HOME}/openshift/e2e-common.sh"

  if [[ $FULL_MESH == true ]]; then
    # TODO: Channels needs more work in the mesh case, so we run only some basic tests for "non channel" based components
    logger.info 'Starting eventing e2e tests for mesh case'
    upstream_knative_eventing_service_mesh_e2e
  else

    cd "${KNATIVE_EVENTING_HOME}"

    # run_e2e_tests defined in knative-eventing
    logger.info 'Starting eventing e2e tests'
    run_e2e_tests

    # run_conformance_tests defined in knative-eventing
    logger.info 'Starting eventing conformance tests'
    run_conformance_tests
  fi
}

function upstream_knative_eventing_service_mesh_e2e {
  go_test_e2e -failfast -timeout=30m -parallel=12 ./test/eventinge2eservicemesh/...
}
