#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

export GO111MODULE=on
# If we run with -mod=vendor here, then generate-groups.sh looks for vendor files in the wrong place.
export GOFLAGS=-mod=

if [ -z "${GOPATH:-}" ]; then
  export GOPATH
  GOPATH=$(go env GOPATH)
fi

REPO_ROOT=$(dirname "${BASH_SOURCE[@]}")/..
CODEGEN_PKG=${CODEGEN_PKG:-$(cd "${REPO_ROOT}"; ls -d -1 "${REPO_ROOT}/vendor/k8s.io/code-generator" 2>/dev/null || echo ../code-generator)}

KNATIVE_CODEGEN_PKG=${KNATIVE_CODEGEN_PKG:-$(cd "${REPO_ROOT}"; ls -d -1 "${REPO_ROOT}/vendor/knative.dev/pkg" 2>/dev/null || echo ../pkg)}

# Generate our own client for Openshift (otherwise injection won't work)
"${CODEGEN_PKG}/generate-groups.sh" "client,informer,lister" \
  github.com/openshift-knative/serverless-operator/pkg/client github.com/openshift/api \
  "route:v1 config:v1" \
  --go-header-file "${REPO_ROOT}/hack/boilerplate/boilerplate.go.txt"

# Knative Injection (for Openshift)
"${KNATIVE_CODEGEN_PKG}/hack/generate-knative.sh" "injection" \
  github.com/openshift-knative/serverless-operator/pkg/client github.com/openshift/api \
  "route:v1 config:v1" \
  --go-header-file "${REPO_ROOT}/hack/boilerplate/boilerplate.go.txt"
