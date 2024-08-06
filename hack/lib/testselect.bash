#!/usr/bin/env bash

function run_testselect {
  if [[ -n "${ARTIFACT_DIR:-}" && -n "${CLONEREFS_OPTIONS:-}" ]]; then
    local clonedir rootdir hack_tmp_dir

    hack_tmp_dir=$(mktemp -d)
    git clone --branch main https://github.com/openshift-knative/hack "$hack_tmp_dir"
    pushd "$hack_tmp_dir" || return $?
    go install github.com/openshift-knative/hack/cmd/testselect
    popd || return $?
    rm -rf "$hack_tmp_dir"

    clonedir=$(mktemp -d)

    # CLONEREFS_OPTIONS var is set in CI
    echo "${CLONEREFS_OPTIONS}" > "${ARTIFACT_DIR}/clonerefs.json"

    cat "${ARTIFACT_DIR}/clonerefs.json"

    rootdir="$(dirname "$(dirname "$(dirname "$(realpath "${BASH_SOURCE[0]}")")")")"

    # The testselect clones a repository. Make sure it's cloned into a temp dir.
    pushd "$clonedir" || return $?
    "$(go env GOPATH)/bin/testselect" --testsuites="${rootdir}/test/testsuites.yaml" --clonerefs="${ARTIFACT_DIR}/clonerefs.json" --output="${ARTIFACT_DIR}/tests.txt"
    popd || return $?

    logger.info 'Tests to be run:'
    cat "${ARTIFACT_DIR}/tests.txt"
  fi
}
