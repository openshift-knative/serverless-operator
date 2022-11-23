#!/usr/bin/env bash

set -Eeuo pipefail

template="${1:?Provide template file as arg[1]}"
target="${2:?Provide a target annotations file as arg[2]}"

# shellcheck disable=SC1091,SC1090
source "$(dirname "${BASH_SOURCE[0]}")/../lib/metadata.bash"

function add_entries {
  cat << EOF | go run github.com/mikefarah/yq/v3@latest write --inplace --script - "$1"
- command: update
  path: entries
  value:
    - name: "${2}"
    - name: "${3}"
      replaces: ${2}
      skipRange: "${4}"
EOF
}

# Start fresh
cp "$template" "$target"

add_entries "$target" \
  "$(metadata.get project.name).v$(metadata.get olm.replaces)" \
  "$(metadata.get project.name).v$(metadata.get project.version)" \
  "$(metadata.get olm.skipRange)"

