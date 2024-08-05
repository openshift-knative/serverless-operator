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
values[LATEST_VERSIONED_CHANNEL]="$(metadata.get 'olm.channels.list[1]')"
values[PREVIOUS_CHANNEL]="$(metadata.get 'olm.channels.list[2]')"
values[PREVIOUS_REPLACES_CHANNEL]="$(metadata.get 'olm.channels.list[3]')"

values[PREVIOUS_CHANNEL_HEAD]="${values[PREVIOUS_CHANNEL]#stable-}.0"
values[PREVIOUS_REPLACES_CHANNEL_HEAD]="${values[PREVIOUS_REPLACES_CHANNEL]#stable-}.0"

# Default channel includes more versions back to allow testing upgrades from older versions.
function add_default_channel_entries {
  local default_entry_yaml default_channel
  default_entry_yaml="$(mktemp -t default-entry-XXXXX.yaml)"
  default_channel=$(metadata.get olm.channels.default)
  cat > "${default_entry_yaml}" <<EOF
schema: olm.channel
name: ${default_channel}
package: serverless-operator
entries:
EOF

  declare -a channels
  channels=($(metadata.get 'olm.channels.list[*]'))
  # Delete first element as this is the default channel
  unset 'channels[0]'
  for i in "${!channels[@]}"; do
    current_version=${channels[$i]#stable-}.0
    previous_i=$(( i+1 ))
    # If this is the last item only enter name, without "replaces".
    if [[ $i -eq ${#channels[@]} ]]; then
      yq write --inplace "$default_entry_yaml" 'entries[+].name' "serverless-operator-v${current_version}"
      break
    fi
    previous_version=${channels[$previous_i]#stable-}.0

  cat << EOF | yq write --inplace --script - "$default_entry_yaml"
- command: update
  path: entries[+]
  value:
    name: "serverless-operator-v${current_version}"
    replaces: "serverless-operator-v${previous_version}"
    skipRange: ">=${previous_version} <${current_version}"
EOF
  done

  echo "---" >> ${1}
  cat "${default_entry_yaml}" >> ${1}
}

# Start fresh
cp "$template" "$target"

for before in "${!values[@]}"; do
  echo "Value: ${before} -> ${values[$before]}"
  sed --in-place "s/__${before}__/${values[${before}]}/" "$target"
done

add_default_channel_entries "${target}"
