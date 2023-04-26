#!/usr/bin/env bash

function run_testselect {
  if [[ -n "${ARTIFACT_DIR:-}" && -n "${CLONEREFS_OPTIONS:-}" ]]; then
    GO111MODULE=off go get github.com/openshift-knative/hack/cmd/testselect

    local clonedir rootdir
    clonedir=$(mktemp -d)

    # CLONEREFS_OPTIONS var is set in CI
    echo "${CLONEREFS_OPTIONS}" > "${ARTIFACT_DIR}/clonerefs.json"

    cat "${ARTIFACT_DIR}/clonerefs.json"

    rootdir="$(dirname "$(dirname "$(dirname "$(realpath "${BASH_SOURCE[0]}")")")")"

    # The testselect clones a repository. Make sure it's cloned into a temp dir.
    pushd "$clonedir" || return $?
    "${GOPATH}/bin/testselect" --testsuites="${rootdir}/test/testsuites.yaml" --clonerefs="${ARTIFACT_DIR}/clonerefs.json" --output="${ARTIFACT_DIR}/tests.txt"
    popd || return $?

    logger.info 'Tests to be run:'
    cat "${ARTIFACT_DIR}/tests.txt"
  fi
}
