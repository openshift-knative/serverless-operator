#!/usr/bin/env bash

readonly BUILD_NUMBER=${BUILD_NUMBER:-$(uuidgen)}

# shellcheck source=vendor/github.com/knative/test-infra/scripts/e2e-tests.sh
source "$(dirname "${BASH_SOURCE[0]}")/../../vendor/github.com/knative/test-infra/scripts/e2e-tests.sh"

readonly KUBECONFIG="${KUBECONFIG:-$(realpath ~/.kube/config)}"
readonly OPENSHIFT_REGISTRY="${OPENSHIFT_REGISTRY:-"registry.svc.ci.openshift.org"}"
readonly INTERNAL_REGISTRY="${INTERNAL_REGISTRY:-"image-registry.openshift-image-registry.svc:5000"}"
readonly TEST_NAMESPACE="${TEST_NAMESPACE:-serverless-tests}"
readonly SERVING_NAMESPACE="${SERVING_NAMESPACE:-knative-serving}"
readonly OPERATORS_NAMESPACE="${OPERATORS_NAMESPACE:-openshift-operators}"
readonly OPERATOR="${OPERATOR:-serverless-operator}"
readonly CATALOG_SOURCE_FILENAME="${CATALOG_SOURCE_FILENAME:-catalogsource-ci.yaml}"
readonly INTERACTIVE="${INTERACTIVE:-$(test -z "${GDMSESSION}"; echo $?)}"
readonly SCALE_UP="${SCALE_UP:-true}"
readonly TEARDOWN="${TEARDOWN:-on_exit}"

# shellcheck source=test/library/bashlang.bash
source "$(dirname ${BASH_SOURCE[0]})/bashlang.bash"
# shellcheck source=test/library/ui.bash
source "$(dirname ${BASH_SOURCE[0]})/ui.bash"
# shellcheck source=test/library/common.bash
source "$(dirname ${BASH_SOURCE[0]})/common.bash"
# shellcheck source=test/library/oc-helpers.bash
source "$(dirname ${BASH_SOURCE[0]})/oc-helpers.bash"
# shellcheck source=test/library/servicemesh.bash
source "$(dirname ${BASH_SOURCE[0]})/servicemesh.bash"

function initialize {
  if [[ "${TEARDOWN}" == "on_exit" ]]; then
    logger.debug 'Registering trap for teardown as EXIT'
    trap teardown EXIT
    return 0
  fi
  if [[ "${TEARDOWN}" == "at_start" ]]; then
    teardown
    return 0
  fi
  logger.error "TEARDOWN should only have a one of values: \"on_exit\", \"at_start\", but given: ${TEARDOWN}."
  return 2
}

function teardown {
  logger.warn "Teardown 💀"
  delete_namespaces
  delete_catalog_source
  delete_users
}

function run_e2e_tests {
  declare -al kubeconfigs
  local kubeconfigs_str
  
  logger.info "Running tests"
  kubeconfigs+=("${KUBECONFIG}")
  for cfg in user*.kubeconfig; do
    kubeconfigs+=("$(pwd)/${cfg}")
  done
  kubeconfigs_str="$(array.join , "${kubeconfigs[@]}")"
  logger.debug "Kubeconfigs: ${kubeconfigs_str}"
  go test -v -tags=e2e -count=1 -timeout=10m -parallel=1 ./test/e2e \
    --kubeconfig "${kubeconfigs_str}" \
    && logger.success 'Tests has passed' && return 0 \
    || logger.error 'Tests have failures!' \
    && return 1
}

