#!/usr/bin/env bash

set -Eeuo pipefail

# shellcheck disable=SC1091,SC1090
source "$(dirname "${BASH_SOURCE[0]}")/lib/__sources__.bash"

snapshot_dir="${1:?Provide the directory containing the override snapshots as arg[1]}"

print_cves "${snapshot_dir}"
