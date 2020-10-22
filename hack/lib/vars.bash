#!/usr/bin/env bash

export BUILD_NUMBER=${BUILD_NUMBER:-$(head -c 128 < /dev/urandom | base64 | fold -w 8 | head -n 1)}

if [[ -n "${ARTIFACT_DIR:-}" ]]; then
  ARTIFACTS="${ARTIFACT_DIR}/build-${BUILD_NUMBER}"
  export ARTIFACTS
  mkdir -p "${ARTIFACTS}"
fi

# shellcheck disable=SC1091,SC1090
source "$(dirname "${BASH_SOURCE[0]}")/../../test/vendor/knative.dev/test-infra/scripts/e2e-tests.sh"

# Adjust these when upgrading the knative versions.
export KNATIVE_SERVING_VERSION="${KNATIVE_SERVING_VERSION:-v$(metadata.get dependencies.serving)}"
export KNATIVE_EVENTING_VERSION="${KNATIVE_EVENTING_VERSION:-v$(metadata.get dependencies.eventing)}"
export KNATIVE_EVENTING_CONTRIB_VERSION="${KNATIVE_EVENTING_CONTRIB_VERSION:-v$(metadata.get dependencies.eventing_contrib)}"

CURRENT_CSV="$(metadata.get project.name).v$(metadata.get project.version)"
PREVIOUS_CSV="$(metadata.get project.name).v$(metadata.get olm.replaces)"
export CURRENT_CSV PREVIOUS_CSV

# Directories below are filled with source code by ci-operator
export KNATIVE_SERVING_HOME="${GOPATH}/src/knative.dev/serving"
export KNATIVE_EVENTING_HOME="${GOPATH}/src/knative.dev/eventing"
export KNATIVE_EVENTING_CONTRIB_HOME="${GOPATH}/src/knative.dev/eventing-contrib"

export CATALOG_SOURCE_FILENAME="${CATALOG_SOURCE_FILENAME:-catalogsource-ci.yaml}"
export DOCKER_REPO_OVERRIDE="${DOCKER_REPO_OVERRIDE:-}"
export INTERACTIVE="${INTERACTIVE:-$(test -z "${GDMSESSION}"; echo $?)}"
export KUBECONFIG="${KUBECONFIG:-$(realpath ~/.kube/config)}"
export OPENSHIFT_CI="${OPENSHIFT_CI:-}"
export OPERATOR="${OPERATOR:-serverless-operator}"
export SCALE_UP="${SCALE_UP:--1}"

export OLM_NAMESPACE="${OLM_NAMESPACE:-openshift-marketplace}"
export OPERATORS_NAMESPACE="${OPERATORS_NAMESPACE:-openshift-serverless}"
export SERVERLESS_NAMESPACE="${SERVERLESS_NAMESPACE:-serverless}"
export SERVING_NAMESPACE="${SERVING_NAMESPACE:-knative-serving}"
export EVENTING_NAMESPACE="${EVENTING_NAMESPACE:-knative-eventing}"
export EVENTING_SOURCES_NAMESPACE="${EVENTING_SOURCES_NAMESPACE:-knative-sources}"
# eventing e2e and conformance tests use a container for tracing tests that has hardcoded `istio-system` in it
export ZIPKIN_NAMESPACE="${ZIPKIN_NAMESPACE:-istio-system}"

declare -a NAMESPACES
NAMESPACES=("${SERVERLESS_NAMESPACE}" "${ZIPKIN_NAMESPACE}" "${OPERATORS_NAMESPACE}")
export NAMESPACES
export UPGRADE_SERVERLESS="${UPGRADE_SERVERLESS:-"true"}"
export UPGRADE_CLUSTER="${UPGRADE_CLUSTER:-"false"}"
# Change this when forcing the upgrade to an image that is not yet available via upgrade channel
export UPGRADE_OCP_IMAGE="${UPGRADE_OCP_IMAGE:-}"

export INSTALL_PREVIOUS_VERSION="${INSTALL_PREVIOUS_VERSION:-"false"}"


# Using first channel on the list, instead of default one
export OLM_CHANNEL="${OLM_CHANNEL:-$(metadata.get 'olm.channels.list[*]' | head -n 1)}"
# Change this when upgrades need switching to a different channel
export OLM_UPGRADE_CHANNEL="${OLM_UPGRADE_CHANNEL:-"$OLM_CHANNEL"}"
export OLM_SOURCE="${OLM_SOURCE:-"$OPERATOR"}"
export TEST_KNATIVE_UPGRADE="${TEST_KNATIVE_UPGRADE:-true}"
export TEST_KNATIVE_E2E="${TEST_KNATIVE_E2E:-true}"
export TEST_KNATIVE_KAFKA="${TEST_KNATIVE_KAFKA:-false}"

# Makefile triggers for modular install
export INSTALL_SERVING="${INSTALL_SERVING:-true}"
export INSTALL_EVENTING="${INSTALL_EVENTING:-true}"
export INSTALL_KAFKA="${INSTALL_KAFKA:-false}"
