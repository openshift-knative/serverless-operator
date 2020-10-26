#!/usr/bin/env bash

readonly ROOT_DIR=$(dirname $0)/..

set -o errexit
set -o nounset
set -o pipefail

# Prune modules.
go mod tidy
go mod vendor

# TODO: Remove this once we bump kubernetes versions.
git apply "$ROOT_DIR/hack/manifestival.patch"

find vendor/ \( -name "OWNERS" \
  -o -name "OWNERS_ALIASES" \
  -o -name "BUILD" \
  -o -name "BUILD.bazel" \
  -o -name "*_test.go" \) -exec rm -fv {} +
