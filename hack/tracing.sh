#!/usr/bin/env bash

# shellcheck disable=SC1091,SC1090
source "$(dirname "${BASH_SOURCE[0]}")/lib/__sources__.bash"

set -Eeuo pipefail

debugging.setup

create_namespaces "${SYSTEM_NAMESPACES[@]}"

if [[ ${UNINSTALL_TRACING:-} == "true" ]]; then
  teardown_tracing
else
  install_tracing
fi
