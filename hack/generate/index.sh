#!/usr/bin/env bash

set -Eeuo pipefail

template="${1:?Provide template file as arg[1]}"
target="${2:?Provide a target annotations file as arg[2]}"

# shellcheck disable=SC1091,SC1090
source "$(dirname "${BASH_SOURCE[0]}")/../lib/metadata.bash"

function add_entries {
  cat << EOF | yq write --inplace --script - "$1"
- command: update
  path: entries
  value:
    - name: "$(metadata.get project.name).v$(metadata.get olm.previous.replaces)"
    - name: "$(metadata.get project.name).v$(metadata.get olm.replaces)"
      replaces: "$(metadata.get project.name).v$(metadata.get olm.previous.replaces)"
      skipRange: "$(metadata.get olm.previous.skipRange)"
    - name: "$(metadata.get project.name).v$(metadata.get project.version)"
      replaces: "$(metadata.get project.name).v$(metadata.get olm.replaces)"
      skipRange: "$(metadata.get olm.skipRange)"
EOF
}

# Start fresh
cp "$template" "$target"

add_entries "$target"
