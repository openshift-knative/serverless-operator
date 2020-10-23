#!/usr/bin/env bash

readonly ROOT_DIR=$(dirname $0)/..
source ${ROOT_DIR}/vendor/knative.dev/test-infra/scripts/library.sh

set -o errexit
set -o nounset
set -o pipefail

cd ${ROOT_DIR}

# This controls the knative release version we track.
KN_VERSION="release-0.17"

# Controls the version of OCP related dependencies.
# We currency stick to 4.4 as 4.5 needs 0.18 K8s clients, which will come via the 0.18
# Knative release.
OCP_VERSION="release-4.4"

# The list of dependencies that we track at HEAD and periodically
# float forward in this repository.
FLOATING_DEPS=(
  # Reenable this once we move to 4.5. Currently hardcoded in go.mod.
  #"github.com/openshift/api@${OCP_VERSION}"
  "github.com/openshift/client-go@${OCP_VERSION}"
  "github.com/operator-framework/operator-lifecycle-manager@${OCP_VERSION}"

  "knative.dev/eventing@${KN_VERSION}"
  "knative.dev/eventing-contrib@${KN_VERSION}"
  "knative.dev/operator@${KN_VERSION}"
  "knative.dev/pkg@${KN_VERSION}"
  "knative.dev/serving@${KN_VERSION}"
  "knative.dev/test-infra@${KN_VERSION}"
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
  go get -d ${FLOATING_DEPS[@]}
fi

# Prune modules.
go mod tidy
go mod vendor

# Remove unnecessary files.
find vendor/ \( -name "OWNERS" -o -name "OWNERS_ALIASES" -o -name "BUILD" -o -name "BUILD.bazel" -o -name "*_test.go" \) -print0 | xargs -0 rm -f

# Add permission for shell scripts
chmod +x $(find vendor -type f -name '*.sh')
