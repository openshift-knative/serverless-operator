#!/usr/bin/env bash

# This script can be used to install KEDA (Custom metrics autoscaler).
#
# shellcheck disable=SC1091,SC1090
source "$(dirname "${BASH_SOURCE[0]}")/lib/__sources__.bash"

set -Eeuo pipefail

if [[ $UNINSTALL_KEDA == "true" ]]; then
  uninstall_keda || exit 1
else
  install_keda || exit 2
fi

exit 0
