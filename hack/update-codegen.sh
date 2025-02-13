#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail


# If we run with -mod=vendor here, then generate-groups.sh looks for vendor files in the wrong place.
export GOFLAGS=-mod=

REPO_ROOT=$(dirname "${BASH_SOURCE[@]}")/..
CODEGEN_PKG=${CODEGEN_PKG:-"${REPO_ROOT}/vendor/k8s.io/code-generator"}

# shellcheck disable=SC1091
source "${REPO_ROOT}"/vendor/knative.dev/hack/codegen-library.sh

KNATIVE_CODEGEN_PKG=${KNATIVE_CODEGEN_PKG:-"${REPO_ROOT}/vendor/knative.dev/pkg"}

# Due to the inherent structure of openshift/client-go packages, that every resource group has its own top-level dir,
# we have to generate Knative injections per group.

# Knative Injection (for Openshift) v1.Route
OUTPUT_PKG="github.com/openshift-knative/serverless-operator/pkg/client/route/injection" \
"${KNATIVE_CODEGEN_PKG}/hack/generate-knative.sh" "injection" \
  github.com/openshift/client-go/route github.com/openshift/api \
  "route:v1" \
  --go-header-file "${REPO_ROOT}/hack/boilerplate/boilerplate.go.txt"

# Knative Injection (for Openshift) v1.Config
OUTPUT_PKG="github.com/openshift-knative/serverless-operator/pkg/client/config/injection" \
"${KNATIVE_CODEGEN_PKG}/hack/generate-knative.sh" "injection" \
  github.com/openshift/client-go/config github.com/openshift/api \
  "config:v1" \
  --go-header-file "${REPO_ROOT}/hack/boilerplate/boilerplate.go.txt"

# In future release 1.18, Knative codegen will have a new option `--plural-exceptions`. Until than we have to live with this sed.
# https://github.com/knative/pkg/pull/3146 
echo "Fix DNS plural form"
find pkg/client/config/injection/informers/config/v1/dns -name "*.go" -exec sed -i 's/DNSs()/DNSes()/g' {} \;
