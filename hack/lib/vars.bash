#!/usr/bin/env bash

readonly BUILD_NUMBER=${BUILD_NUMBER:-$(uuidgen)}

# shellcheck disable=SC1091,SC1090
source "$(dirname "${BASH_SOURCE[0]}")/../../test/vendor/github.com/knative/test-infra/scripts/e2e-tests.sh"

readonly KNATIVE_VERSION="${KNATIVE_VERSION:-v0.13.1}"

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

declare -a NAMESPACES
NAMESPACES=("${SERVING_NAMESPACE}" "${SERVERLESS_NAMESPACE}")
export NAMESPACES
readonly UPGRADE_SERVERLESS="${UPGRADE_SERVERLESS:-"true"}"
readonly UPGRADE_CLUSTER="${UPGRADE_CLUSTER:-"false"}"

readonly INSTALL_PREVIOUS_VERSION="${INSTALL_PREVIOUS_VERSION:-"false"}"
readonly CHANNEL="${CHANNEL:-"preview-4.3"}"
