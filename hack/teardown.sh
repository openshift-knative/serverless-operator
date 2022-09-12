#!/usr/bin/env bash

# shellcheck disable=SC1091,SC1090
source "$(dirname "${BASH_SOURCE[0]}")/lib/__sources__.bash"

set -Eeuo pipefail

debugging.setup

teardown_serverless
teardown_tracing
uninstall_mesh
delete_catalog_source
delete_namespaces "${SYSTEM_NAMESPACES[@]}" "${TEST_NAMESPACES[@]}"
