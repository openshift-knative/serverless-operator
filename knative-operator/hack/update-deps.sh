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

rm -rf $(find vendor/ -name 'OWNERS')
# Remove unit tests & e2e tests.
rm -rf $(find vendor/ -path '*/pkg/*_test.go')
rm -rf $(find vendor/ -path '*/e2e/*_test.go')

# Add permission for shell scripts
chmod +x $(find vendor -type f -name '*.sh')