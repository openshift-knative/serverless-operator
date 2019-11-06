#!/usr/bin/env bash

readonly BUILD_NUMBER=${BUILD_NUMBER:-$(uuidgen)}

# shellcheck disable=SC1091,SC1090
source "$(dirname "${BASH_SOURCE[0]}")/../../vendor/github.com/knative/test-infra/scripts/e2e-tests.sh"

readonly KUBECONFIG="${KUBECONFIG:-$(realpath ~/.kube/config)}"
readonly OPENSHIFT_REGISTRY="${OPENSHIFT_REGISTRY:-"registry.svc.ci.openshift.org"}"
readonly INTERNAL_REGISTRY="${INTERNAL_REGISTRY:-"image-registry.openshift-image-registry.svc:5000"}"
readonly SERVING_NAMESPACE="${SERVING_NAMESPACE:-knative-serving}"
readonly OPERATORS_NAMESPACE="${OPERATORS_NAMESPACE:-openshift-operators}"
readonly SERVERLESS_NAMESPACE="${SERVERLESS_NAMESPACE:-serverless}"
readonly OPERATOR="${OPERATOR:-serverless-operator}"
readonly CATALOG_SOURCE_FILENAME="${CATALOG_SOURCE_FILENAME:-catalogsource-ci.yaml}"
readonly INTERACTIVE="${INTERACTIVE:-$(test -z "${GDMSESSION}"; echo $?)}"
readonly SCALE_UP="${SCALE_UP:-6}"
declare -a NAMESPACES
NAMESPACES=("${SERVING_NAMESPACE}" "${SERVERLESS_NAMESPACE}")
export NAMESPACES
