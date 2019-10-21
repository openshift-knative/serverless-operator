#!/usr/bin/env bash

# shellcheck source=test/library/loader.bash
source "$(dirname ${BASH_SOURCE[0]})/loader.bash"

loader_flag "${BASH_SOURCE[0]}"
loader_addpath "$(dirname "${BASH_SOURCE[0]}")"

include logic/lifecycle.bash
include logic/tests.bash

include infra/ocp/catalogsource.bash
include infra/ocp/dump.bash
include infra/ocp/namespaces.bash
include infra/ocp/scaleup.bash
include infra/ocp/servicemesh.bash
include infra/ocp/users.bash

loader_finish
