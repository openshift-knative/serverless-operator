#!/usr/bin/env bash

readonly BUILD_NUMBER=${BUILD_NUMBER:-$(head -c 128 < /dev/urandom | base64 | fold -w 8 | head -n 1)}

if [[ -n "${ARTIFACT_DIR:-}" ]]; then
  ARTIFACTS="${ARTIFACT_DIR}/build-${BUILD_NUMBER}"
  readonly ARTIFACTS
  mkdir -p "${ARTIFACTS}"
fi

# shellcheck disable=SC1091,SC1090
source "$(dirname "${BASH_SOURCE[0]}")/../../test/vendor/knative.dev/test-infra/scripts/e2e-tests.sh"

readonly KNATIVE_SERVING_VERSION="${KNATIVE_SERVING_VERSION:-v0.13.3}"
readonly KNATIVE_SERVING_OPERATOR_VERSION="${KNATIVE_SERVING_OPERATOR_VERSION:-v0.13.3}"
# KNATIVE_SEVING_HOME is filled with Knative Serving sources by ci-operator
readonly KNATIVE_SERVING_HOME="${GOPATH}/src/knative.dev/serving"
readonly KNATIVE_EVENTING_VERSION="${KNATIVE_EVENTING_VERSION:-v0.13.0}"
readonly KNATIVE_EVENTING_OPERATOR_VERSION="${KNATIVE_EVENTING_OPERATOR_VERSION:-v0.13.3}"

readonly KNATIVE_SERVING_BRANCH="${KNATIVE_SERVING_BRANCH:-release-${KNATIVE_SERVING_VERSION}}"
readonly KNATIVE_SERVING_OPERATOR_BRANCH="${KNATIVE_SERVING_OPERATOR_BRANCH:-openshift-${KNATIVE_SERVING_OPERATOR_VERSION}}"
readonly KNATIVE_SERVING_REPO="${KNATIVE_SERVING_REPO:-"https://github.com/openshift/knative-serving.git"}"
readonly KNATIVE_SERVING_OPERATOR_REPO="${KNATIVE_SERVING_OPERATOR_REPO:-"https://github.com/openshift-knative/serving-operator.git"}"
readonly KNATIVE_EVENTING_BRANCH="${KNATIVE_EVENTING_BRANCH:-release-${KNATIVE_EVENTING_VERSION}}"
readonly KNATIVE_EVENTING_OPERATOR_BRANCH="${KNATIVE_EVENTING_OPERATOR_BRANCH:-openshift-${KNATIVE_EVENTING_OPERATOR_VERSION}}"
readonly KNATIVE_EVENTING_REPO="${KNATIVE_EVENTING_REPO:-"https://github.com/openshift/knative-eventing.git"}"
readonly KNATIVE_EVENTING_OPERATOR_REPO="${KNATIVE_EVENTING_OPERATOR_REPO:-"https://github.com/openshift-knative/eventing-operator.git"}"

readonly CATALOG_SOURCE_FILENAME="${CATALOG_SOURCE_FILENAME:-catalogsource-ci.yaml}"
readonly DOCKER_REPO_OVERRIDE="${DOCKER_REPO_OVERRIDE:-}"
readonly INTERACTIVE="${INTERACTIVE:-$(test -z "${GDMSESSION}"; echo $?)}"
readonly KUBECONFIG="${KUBECONFIG:-$(realpath ~/.kube/config)}"
readonly OPENSHIFT_BUILD_NAMESPACE="${OPENSHIFT_BUILD_NAMESPACE:-}"
readonly OPERATOR="${OPERATOR:-serverless-operator}"
readonly SCALE_UP="${SCALE_UP:-6}"

readonly OLM_NAMESPACE="${OLM_NAMESPACE:-openshift-marketplace}"
readonly OPERATORS_NAMESPACE="${OPERATORS_NAMESPACE:-openshift-operators}"
readonly SERVERLESS_NAMESPACE="${SERVERLESS_NAMESPACE:-serverless}"
readonly SERVING_NAMESPACE="${SERVING_NAMESPACE:-knative-serving}"
readonly EVENTING_NAMESPACE="${EVENTING_NAMESPACE:-knative-eventing}"

declare -a NAMESPACES
NAMESPACES=("${SERVING_NAMESPACE}" "${SERVERLESS_NAMESPACE}" "${EVENTING_NAMESPACE}")
export NAMESPACES
readonly UPGRADE_SERVERLESS="${UPGRADE_SERVERLESS:-"true"}"
readonly UPGRADE_CLUSTER="${UPGRADE_CLUSTER:-"false"}"

readonly INSTALL_PREVIOUS_VERSION="${INSTALL_PREVIOUS_VERSION:-"false"}"
export OLM_CHANNEL="${OLM_CHANNEL:-"preview-4.6"}"
# Change this when upgrades need switching to a different channel
export OLM_UPGRADE_CHANNEL="${OLM_UPGRADE_CHANNEL:-"$OLM_CHANNEL"}"
export OLM_SOURCE="${OLM_SOURCE:-"$OPERATOR"}"
