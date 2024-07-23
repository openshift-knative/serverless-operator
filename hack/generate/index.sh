#!/usr/bin/env bash

set -Eeuo pipefail

template="${1:?Provide template file as arg[1]}"
target="${2:?Provide a target annotations file as arg[2]}"

# shellcheck disable=SC1091,SC1090
source "$(dirname "${BASH_SOURCE[0]}")/../lib/metadata.bash"

declare -A values

values[VERSION]="$(metadata.get project.version)"
values[PREVIOUS_VERSION]="$(metadata.get olm.replaces)"
values[PREVIOUS_REPLACES]="$(metadata.get olm.previous.replaces)"
values[DEFAULT_CHANNEL]="$(metadata.get olm.channels.default)"
values[LATEST_VERSIONED_CHANNEL]="$(metadata.get 'olm.channels.list[*]' | head -n 2 | tail -n 1)"
values[PREVIOUS_CHANNEL]="$(metadata.get 'olm.channels.list[*]' | head -n 3 | tail -n 1)"
values[PREVIOUS_REPLACES_CHANNEL]="$(metadata.get 'olm.channels.list[*]' | head -n 4 | tail -n 1)"

values[PREVIOUS_CHANNEL_HEAD]="${values[PREVIOUS_CHANNEL]#stable-}.0"
values[PREVIOUS_REPLACES_CHANNEL_HEAD]="${values[PREVIOUS_REPLACES_CHANNEL]#stable-}.0"

# Start fresh
cp "$template" "$target"

for before in "${!values[@]}"; do
  echo "Value: ${before} -> ${values[$before]}"
  sed --in-place "s/__${before}__/${values[${before}]}/" "$target"
done
