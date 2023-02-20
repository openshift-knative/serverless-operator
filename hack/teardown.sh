#!/usr/bin/env bash

# shellcheck disable=SC1091,SC1090
source "$(dirname "${BASH_SOURCE[0]}")/lib/__sources__.bash"

set -Eeuo pipefail

debugging.setup
dump_state.setup

teardown_serverless
delete_catalog_source
teardown_tracing
delete_namespaces "${SYSTEM_NAMESPACES[@]}"
