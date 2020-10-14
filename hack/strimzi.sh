#!/usr/bin/env bash

# This script can be used to install Strimzi and create a Kafka instance on cluster.
#
# shellcheck disable=SC1091,SC1090
source "$(dirname "${BASH_SOURCE[0]}")/lib/__sources__.bash"

set -Eeuo pipefail

if [[ $UNINSTALL_STRIMZI == "true" ]]; then
  uninstall_strimzi || exit 1
else
  install_strimzi || exit 2
fi

exit 0
