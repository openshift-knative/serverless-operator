#!/usr/bin/env bash

function clone_and_build_testselect {
  local hack
  hack=$(mktemp -d)
  git clone --branch select_testsuites https://github.com/mgencur/hack "$hack"
  pushd "$hack" || return
  go install ./cmd/testselect
  popd || return
}

if [[ -n "${ARTIFACT_DIR:-}" ]]; then
  # TODO: Remove when testselect is available in github.com/openshift-knative/hack
  # Then we can just call go run github.com/openshift-knative/hack/cmd/testselect
  clone_and_build_testselect

  # CLONEREFS_OPTIONS var is set in CI
  echo "${CLONEREFS_OPTIONS}" > "${ARTIFACT_DIR}/clonerefs.json"

  cat "${ARTIFACT_DIR}/clonerefs.json"

  rootdir="$(dirname "$(dirname "$(dirname "$(realpath "${BASH_SOURCE[0]}")")")")"

  testselect --testsuites="${rootdir}/test/testsuites.yaml" --clonerefs="${ARTIFACT_DIR}/clonerefs.json" --output="${ARTIFACT_DIR}/tests.txt"

  logger.info 'Tests to be run:'
  cat "${ARTIFACT_DIR}/tests.txt"
fi
