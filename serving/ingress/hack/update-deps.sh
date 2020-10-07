#!/usr/bin/env bash

readonly ROOT_DIR=$(dirname $0)/..
source ${ROOT_DIR}/vendor/knative.dev/test-infra/scripts/library.sh

set -o errexit
set -o nounset
set -o pipefail

cd ${ROOT_DIR}

# This controls the knative release version we track.
KN_VERSION="release-0.17" # This is for controlling the knative related release version.

# The list of dependencies that we track at HEAD and periodically
# float forward in this repository.
FLOATING_DEPS=(
  "knative.dev/networking@${KN_VERSION}"
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

rm -rf $(find vendor/ -name 'OWNERS')
# Remove unit tests & e2e tests.
rm -rf $(find vendor/ -path '*/pkg/*_test.go')
rm -rf $(find vendor/ -path '*/e2e/*_test.go')

# Add permission for shell scripts
chmod +x $(find vendor -type f -name '*.sh')
