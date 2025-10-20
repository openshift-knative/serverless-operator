#!/usr/bin/env bash

set -Eeuo pipefail

target="${1:?Provide a target index yaml file as arg[1]}"

# shellcheck disable=SC1091,SC1090
source "$(dirname "${BASH_SOURCE[0]}")/../lib/__sources__.bash"

function add_channel_entries {
  local channel_entry_yaml channel target version replaces_version skip_range previous_bundles_len
  channel=${1:?Provide channel name}
  target=${2:?Provide target file name}
  channel_entry_yaml="$(mktemp -t default-entry-XXXXX.yaml)"

  # Initialize a temporary file for a single entry.
  cat > "${channel_entry_yaml}" <<EOF
schema: olm.channel
name: ${channel}
package: serverless-operator
entries:
EOF

  version=$(metadata.get 'project.version')
  replaces_version=$(metadata.get 'olm.replaces')
  skip_range=$(metadata.get 'olm.skipRange')

  cat << EOF | yq write --inplace --script - "$channel_entry_yaml"
  - command: update
    path: entries[+]
    value:
      name: "serverless-operator.v${version}"
      replaces: "serverless-operator.v${replaces_version}"
      skipRange: "${skip_range}"
EOF

  previous_bundles_len=$(metadata.get 'olm.previousBundles' | yq read - -l '')
  for i in $(seq ${previous_bundles_len}); do
    version=$(metadata.get "olm.previousBundles | yq read - '[$i]['version']")
    replaces_version=$(metadata.get "olm.previousBundles | yq read - '[$i]['replaces']")
    skip_range=$(metadata.get "olm.previousBundles | yq read - '[$i]['skipRange']")

    if [[ $i -eq $previous_bundles_len ]]; then
      yq write --inplace "$channel_entry_yaml" 'entries[+].name' "serverless-operator.v${version}"
    else
      cat << EOF | yq write --inplace --script - "$channel_entry_yaml"
  - command: update
    path: entries[+]
    value:
      name: "serverless-operator.v${version}"
      replaces: "serverless-operator.v${replaces_version}"
      skipRange: "${skip_range}"
EOF
    fi
  done

  echo "---" >> "${target}"
  cat "${channel_entry_yaml}" >> "${target}"
}

# Clear the file.
rm -f "${target}"

while IFS=$'\n' read -r channel; do
  add_channel_entries "$channel" "${target}"
done < <(metadata.get 'olm.channels.list[*]')
