#!/usr/bin/env bash

# Define the files to exclude
readonly EXCLUDE_FILES=(
  'olm-catalog/serverless-operator/manifests/serverless-operator.clusterserviceversion.yaml'
  'olm-catalog/serverless-operator-index/Dockerfile'
  'test/images-rekt.yaml'
)
# Define the patterns to exclude
readonly EXCLUDE_PATTERNS=(
  '*sha256:*'
)

# Function to check if a file should be excluded
function should_exclude() {
  local file="$1"
  for pattern in "${EXCLUDE_PATTERNS[@]}"; do
    diff="$(git --no-pager -c color.ui=never diff --unified=0 "$file" | grep '^[+-][\ a-z]')"
    while IFS= read -r line; do
      # shellcheck disable=SC2053
      if  [[ $line == $pattern || $line =~ $pattern ]]; then
        echo "Excluding line $line since matches pattern $pattern"
      else
        echo "line '$line' doesn't match pattern '$pattern', failing the exclude check"
        return 1
      fi
    done <<< "$diff"
  done

  return 0
}

# shellcheck disable=SC2016
function debug_log_fail() {
  echo '::debug::Running `git status`'
  git -c color.status=always status
  echo '::debug::Running `git diff`'
  git --no-pager -c color.ui=always diff
  echo '::error::Not all generated files are committed. Run `make generated-files` and commit files.'
  echo '::warning::`make generated-files` needs to be run on GOPATH due to https://github.com/knative/pkg/issues/1287'
}

# shellcheck disable=SC2143
if [ -n "$(git status --porcelain | grep -v -E "$(IFS=\|; echo "${EXCLUDE_FILES[*]}")")" ]; then
  debug_log_fail
  exit 33
fi

# shellcheck disable=SC2143
if [ -n "$(git status --porcelain | grep -E "$(IFS=\|; echo "${EXCLUDE_FILES[*]}")")" ]; then
  echo 'Excluded files are different'

  git diff --name-only | while read -r file; do
    if ! should_exclude "$file"; then
      git --no-pager -c color.ui=always diff "$file"
      debug_log_fail
      exit 33
    fi
  done
fi
