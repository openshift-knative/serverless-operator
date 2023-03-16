#!/usr/bin/env bash

function run_testselect {
  if [[ -n "${ARTIFACT_DIR:-}" ]]; then
    GO111MODULE=off go get github.com/openshift-knative/hack/cmd/testselect

    # CLONEREFS_OPTIONS var is set in CI
    echo "${CLONEREFS_OPTIONS}" > "${ARTIFACT_DIR}/clonerefs.json"

    cat "${ARTIFACT_DIR}/clonerefs.json"

    local rootdir
    rootdir="$(dirname "$(dirname "$(dirname "$(realpath "${BASH_SOURCE[0]}")")")")"

    "${GOPATH}/bin/testselect" --testsuites="${rootdir}/test/testsuites.yaml" --clonerefs="${ARTIFACT_DIR}/clonerefs.json" --output="${ARTIFACT_DIR}/tests.txt"

    logger.info 'Tests to be run:'
    cat "${ARTIFACT_DIR}/tests.txt"
  fi
}
