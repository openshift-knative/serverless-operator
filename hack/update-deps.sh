#!/usr/bin/env bash

ROOT_DIR=$(dirname "$0")/..
readonly ROOT_DIR

# shellcheck disable=SC1091,SC1090
source "${ROOT_DIR}/vendor/knative.dev/hack/library.sh"
# shellcheck disable=SC1091,SC1090
source "${ROOT_DIR}/hack/lib/metadata.bash"

set -o errexit
set -o nounset
set -o pipefail

cd "${ROOT_DIR}"

# This controls the knative release version we track.
KN_VERSION="release-1.15"
EVENTING_VERSION="release-v1.15"
EVENTING_KAFKA_BROKER_VERSION="release-v1.15"
SERVING_VERSION="release-v1.15"
GO_VERSION="$(metadata.get requirements.golang)"
OCP_VERSION="$(metadata.get requirements.ocpVersion.min)"

# The list of dependencies that we track at HEAD and periodically
# float forward in this repository.
FLOATING_DEPS=(
  "knative.dev/operator@${KN_VERSION}"
)

REPLACE_DEPS=(
  "knative.dev/pkg=knative.dev/pkg@${KN_VERSION}"
  "knative.dev/hack=knative.dev/hack@${KN_VERSION}"
  "knative.dev/networking=knative.dev/networking@${KN_VERSION}"
  "knative.dev/reconciler-test=knative.dev/reconciler-test@${KN_VERSION}"
  "knative.dev/eventing=github.com/openshift-knative/eventing@${EVENTING_VERSION}"
  "knative.dev/serving=github.com/openshift-knative/serving@${SERVING_VERSION}"
  "knative.dev/eventing-kafka-broker=github.com/openshift-knative/eventing-kafka-broker@${EVENTING_KAFKA_BROKER_VERSION}"
  "github.com/openshift/api=github.com/openshift/api@release-${OCP_VERSION}"
  "github.com/openshift/client-go=github.com/openshift/client-go@release-${OCP_VERSION}"
  "github.com/openshift/machine-config-operator=github.com/openshift/machine-config-operator@release-${OCP_VERSION}"
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
  export GOPROXY="https://proxy.golang.org,direct"
  # Treat forks specifically due to https://github.com/golang/go/issues/32721
  for dep in "${REPLACE_DEPS[@]}"; do
    go mod edit -replace "${dep}"
    # Let the dependency update the magic SHA otherwise the
    # following "go mod edit" will fail.
    go mod tidy -compat="${GO_VERSION}"
    go mod vendor
  done
  go get -d "${FLOATING_DEPS[@]}"
fi

# Prune modules.
go mod tidy -compat="${GO_VERSION}"
go mod vendor

# Remove unnecessary files.
find vendor/ \( -name "OWNERS" \
  -o -name "OWNERS_ALIASES" \
  -o -name "BUILD" \
  -o -name "BUILD.bazel" \
  -o -name "*_test.go" \) -exec rm -fv {} +

find vendor -type f -name '*.sh' -exec chmod +x {} +

# Apply patches
git apply "${ROOT_DIR}"/hack/patches/*
