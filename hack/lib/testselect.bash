#!/usr/bin/env bash

function clone_and_build_testselect {
  git clone --branch select_testsuites https://github.com/mgencur/hack
  pushd hack || return
  go install ./cmd/testselect
  popd || return
}

if [[ -n "${ARTIFACT_DIR:-}" ]]; then
  # TODO: Remove when testselect is available in github.com/openshift-knative/hack
  # Then we can just call go run github.com/openshift-knative/hack/cmd/testselect
  clone_and_build_testselect

  # CLONEREFS_OPTIONS var is set in CI
  echo "${CLONEREFS_OPTIONS}" > "${ARTIFACTS_DIR}/clonerefs.json"

  cat "${ARTIFACTS_DIR}/clonerefs.json"

  rootdir="$(dirname "$(dirname "$(dirname "$(realpath "${BASH_SOURCE[0]}")")")")"

  testselect --testsuites="${rootdir}/test/testsuites.yaml" --clonerefs="$clonerefs" --output="${ARTIFACTS_DIR}/tests.txt"

  cat "${ARTIFACTS_DIR}/tests.txt"
fi
