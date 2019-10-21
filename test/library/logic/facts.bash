#!/usr/bin/env bash

readonly BUILD_NUMBER=${BUILD_NUMBER:-$(uuidgen)}

# shellcheck source=vendor/github.com/knative/test-infra/scripts/e2e-tests.sh
source "$(dirname "${BASH_SOURCE[0]}")/../../../vendor/github.com/knative/test-infra/scripts/e2e-tests.sh"

readonly KUBECONFIG="${KUBECONFIG:-$(realpath ~/.kube/config)}"
readonly OPENSHIFT_REGISTRY="${OPENSHIFT_REGISTRY:-"registry.svc.ci.openshift.org"}"
readonly INTERNAL_REGISTRY="${INTERNAL_REGISTRY:-"image-registry.openshift-image-registry.svc:5000"}"
readonly TEST_NAMESPACE=serverless-tests
readonly SERVING_NAMESPACE=knative-serving
readonly OPERATORS_NAMESPACE="openshift-operators"
readonly OPERATOR="serverless-operator"
readonly CATALOG_SOURCE_FILENAME="catalogsource-ci.yaml"
readonly CI="${CI:-$(test -z "${GDMSESSION}"; echo $?)}"
readonly SCALE_UP="${SCALE_UP:-true}"
readonly TEARDOWN="${TEARDOWN:-on_exit}"
