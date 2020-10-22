#!/usr/bin/env bash

function upstream_knative_eventing_e2e {
  logger.info 'Running eventing tests'

  export TEST_IMAGE_TEMPLATE="registry.svc.ci.openshift.org/openshift/knative-${KNATIVE_EVENTING_VERSION}:knative-eventing-test-{{.Name}}"

  cd "$KNATIVE_EVENTING_HOME" || return $?

  # shellcheck disable=SC1090
  source "${KNATIVE_EVENTING_HOME}/openshift/e2e-common.sh"

  # run_e2e_tests defined in knative-eventing
  run_e2e_tests
}

function downstream_eventing_e2e_tests {
  declare -a kubeconfigs
  local kubeconfigs_str

  logger.info "Running Eventing downstream tests"
  kubeconfigs+=("${KUBECONFIG}")
  for cfg in user*.kubeconfig; do
    kubeconfigs+=("$(pwd)/${cfg}")
  done
  kubeconfigs_str="$(array.join , "${kubeconfigs[@]}")"

  go_test_e2e -failfast -timeout=30m -parallel=1 ./test/eventinge2e \
    --kubeconfig "${kubeconfigs[0]}" \
    --kubeconfigs "${kubeconfigs_str}" \
    "$@"
}
