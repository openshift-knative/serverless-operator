#!/usr/bin/env bash

set -Eeuo pipefail

# shellcheck disable=SC1091,SC1090
source "$(dirname "${BASH_SOURCE[0]}")/../lib/__sources__.bash"

target_dir="${1:?Provide a target directory for the override snapshots as arg[1]}"

create_component_snapshot "${target_dir}"
create_fbc_snapshots "${target_dir}"
