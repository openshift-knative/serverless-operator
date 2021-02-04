#!/usr/bin/env bash

readonly ROOT_DIR=$(dirname "$0")/..

# shellcheck disable=SC1091,SC1090
source "${ROOT_DIR}/vendor/knative.dev/hack/library.sh"

set -o errexit
set -o nounset
set -o pipefail

cd "${ROOT_DIR}"

# This controls the knative release version we track.
KN_VERSION="release-0.20"

# Controls the version of OCP related dependencies.
OCP_VERSION="release-4.7"

# The list of dependencies that we track at HEAD and periodically
# float forward in this repository.
FLOATING_DEPS=(
  "github.com/openshift/api@${OCP_VERSION}"
  "github.com/openshift/client-go@${OCP_VERSION}"
  "github.com/operator-framework/operator-lifecycle-manager@${OCP_VERSION}"

  "knative.dev/eventing-kafka@${KN_VERSION}"
  "knative.dev/eventing@${KN_VERSION}"
  "knative.dev/hack@${KN_VERSION}"
  "knative.dev/networking@${KN_VERSION}"
  "knative.dev/operator@${KN_VERSION}"
  "knative.dev/pkg@${KN_VERSION}"
  "knative.dev/serving@${KN_VERSION}"
)

# Parse flags to determine if we need to update our floating deps.
GO_GET=0
while [[ $# -ne 0 ]]; do
  parameter=$1
  case ${parameter} in
    --upgrade) GO_GET=1 ;;
    *) abort "unknown option ${parameter}" ;;
  esac
  shift
done
readonly GO_GET

if (( GO_GET )); then
  go get -d "${FLOATING_DEPS[@]}"
fi

# Prune modules.
go mod tidy
go mod vendor

# Remove files conflicting due to logr.
rm -f vendor/knative.dev/pkg/test/logging/tlogger.go \
  vendor/knative.dev/pkg/test/logging/zapr.go \
  vendor/knative.dev/pkg/test/logging/sugar.go \
  vendor/knative.dev/pkg/test/logging/error.go \
  vendor/knative.dev/pkg/test/logging/spew_encoder.go \
  vendor/knative.dev/pkg/test/logging/memory_encoder.go \
  vendor/knative.dev/pkg/test/logging/logger.go

# Remove unnecessary files.
find vendor/ \( -name "OWNERS" \
  -o -name "OWNERS_ALIASES" \
  -o -name "BUILD" \
  -o -name "BUILD.bazel" \
  -o -name "*_test.go" \) -exec rm -fv {} +

find vendor -type f -name '*.sh' -exec chmod +x {} +
