#!/usr/bin/env bash

set -Eeuo pipefail

root="$(dirname "${BASH_SOURCE[0]}")/../../.."

git apply "$root/olm-catalog/serverless-operator/hack/006-operator-vendor.patch"
