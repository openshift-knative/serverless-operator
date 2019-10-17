#!/usr/bin/env bash

REPO_DIR=$(dirname "$(dirname "$(realpath "${BASH_SOURCE[0]}")")")

# shellcheck source=test/library/main.bash
source "${REPO_DIR}/test/library/main.bash"

set -Eeuo pipefail

initialize || exit $?

scale_up_workers || exit 1

create_namespaces || exit 1

create_htpasswd_users && add_roles || exit 1

failed=0

(( !failed )) && ensure_service_mesh_installed || failed=1

(( !failed )) && install_catalogsource || failed=1

(( !failed )) && logger.success 'Cluster prepared for testing.' && run_e2e_tests || failed=1

(( failed )) && dump_state

(( failed )) && exit 1

success
