#!/usr/bin/env bash

# This script can be used to install cert-manager operator.
#
# shellcheck disable=SC1091,SC1090
source "$(dirname "${BASH_SOURCE[0]}")/lib/__sources__.bash"

set -Eeuo pipefail

if [[ $UNINSTALL_CERTMANAGER == "true" ]]; then
  uninstall_certmanager || exit 1
else
  install_certmanager || exit 2
fi

exit 0
