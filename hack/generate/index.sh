#!/usr/bin/env bash

set -Eeuo pipefail

template="${1:?Provide template file as arg[1]}"
target="${2:?Provide a target annotations file as arg[2]}"

# shellcheck disable=SC1091,SC1090
source "$(dirname "${BASH_SOURCE[0]}")/../lib/metadata.bash"
CHANNEL_LIST="$(metadata.get .olm.channels.list[])"
function add_entries {
#   cat << EOF | yq write --inplace --script - "$1"
# - command: update
#   path: entries
#   value:
#     - name: "$(metadata.get .project.name).v$(metadata.get .olm.previous.replaces)"
#     - name: "$(metadata.get .project.name).v$(metadata.get .olm.replaces)"
#       replaces: "$(metadata.get .project.name).v$(metadata.get .olm.previous.replaces)"
#       skipRange: "$(metadata.get .olm.previous.skipRange)"
#     - name: "$(metadata.get .project.name).v$(metadata.get .project.version)"
#       replaces: "$(metadata.get .project.name).v$(metadata.get .olm.replaces)"
#       skipRange: "$(metadata.get .olm.skipRange)"
# EOF
  yq e --inplace ".entries += [] | 
  .entries += {\"name\": \"$(metadata.get .project.name).v$(metadata.get .olm.previous.replaces)\"} | . style=\"double\" |
  .entries += {\"name\": \"$(metadata.get .project.name).v$(metadata.get .olm.replaces)\", \"replaces\": \"$(metadata.get .project.name).v$(metadata.get .olm.previous.replaces)\", \"skipRange\": \"$(metadata.get .olm.previous.skipRange)\"} |
  .entries += {\"name\":  \"$(metadata.get .project.name).v$(metadata.get .project.version)\", \"replaces\": \"$(metadata.get .project.name).v$(metadata.get .olm.replaces)\", \"skipRange\": \"$(metadata.get .olm.skipRange)\"}" "$1"
}

# Start fresh
cp "$template" "$target"
OUTPUT=""
for NAME in $CHANNEL_LIST; do
  tmpfile=$(mktemp)
  sed "s/__CHANNEL__/$NAME/g" "$target" > "$tmpfile"
  add_entries "$tmpfile"
  # if [ -n "$OUTPUT" ]; then
  #   OUTPUT=$OUTPUT$'\n'"---"$'\n'
  #else
    # First line of the file
    #OUTPUT=$"---"$'\n'
  #fi
  OUTPUT=$OUTPUT$(cat "$tmpfile")'\n'
done
rm "$tmpfile"
echo -e "$OUTPUT" > "$target"
