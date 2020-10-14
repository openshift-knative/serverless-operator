#!/usr/bin/env bash

# shellcheck disable=SC1091,SC1090
source "$(dirname "${BASH_SOURCE[0]}")/lib/__sources__.bash"

set -Eeuo pipefail

exitcode=0

(( !exitcode )) && teardown_serverless || exitcode=2
(( !exitcode )) && teardown_tracing || exitcode=3
(( !exitcode )) && delete_catalog_source || exitcode=4
(( !exitcode )) && delete_namespaces || exitcode=5

exit $exitcode
