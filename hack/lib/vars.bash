#!/usr/bin/env bash

readonly BUILD_NUMBER=${BUILD_NUMBER:-$(uuidgen)}

# shellcheck disable=SC1091,SC1090
source "$(dirname "${BASH_SOURCE[0]}")/../../vendor/github.com/knative/test-infra/scripts/e2e-tests.sh"

readonly CATALOG_SOURCE_FILENAME="${CATALOG_SOURCE_FILENAME:-catalogsource-ci.yaml}"
readonly DOCKER_REPO_OVERRIDE="${DOCKER_REPO_OVERRIDE:-}"
readonly INTERACTIVE="${INTERACTIVE:-$(test -z "${GDMSESSION}"; echo $?)}"
readonly KUBECONFIG="${KUBECONFIG:-$(realpath ~/.kube/config)}"
readonly OPERATOR="${OPERATOR:-serverless-operator}"
readonly SCALE_UP="${SCALE_UP:-6}"

readonly OPERATORS_NAMESPACE="${OPERATORS_NAMESPACE:-openshift-operators}"
readonly SERVERLESS_NAMESPACE="${SERVERLESS_NAMESPACE:-serverless}"
readonly SERVING_NAMESPACE="${SERVING_NAMESPACE:-knative-serving}"

declare -a NAMESPACES
NAMESPACES=("${SERVING_NAMESPACE}" "${SERVERLESS_NAMESPACE}")
export NAMESPACES
