#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail


# If we run with -mod=vendor here, then generate-groups.sh looks for vendor files in the wrong place.
export GOFLAGS=-mod=

REPO_ROOT=$(dirname "${BASH_SOURCE[@]}")/..
CODEGEN_PKG=${CODEGEN_PKG:-"${REPO_ROOT}/vendor/k8s.io/code-generator"}

# shellcheck disable=SC1091
source "${CODEGEN_PKG}/kube_codegen.sh"

# shellcheck disable=SC1091
source "${REPO_ROOT}"/vendor/knative.dev/hack/codegen-library.sh

KNATIVE_CODEGEN_PKG=${KNATIVE_CODEGEN_PKG:-"${REPO_ROOT}/vendor/knative.dev/pkg"}

# Generate our own client for Openshift (otherwise injection won't work)
kube::codegen::gen_client \
  --boilerplate "${REPO_ROOT_DIR}/hack/boilerplate/boilerplate.go.txt" \
  --output-dir "${REPO_ROOT_DIR}/pkg/client" \
  --output-pkg "github.com/openshift/api" \
  --with-watch \
  "${REPO_ROOT_DIR}/pkg/apis"

# Knative Injection (for Openshift)
"${KNATIVE_CODEGEN_PKG}/hack/generate-knative.sh" "injection" \
  github.com/openshift-knative/serverless-operator/pkg/client github.com/openshift/api \
  "route:v1 config:v1" \
  --go-header-file "${REPO_ROOT}/hack/boilerplate/boilerplate.go.txt"
