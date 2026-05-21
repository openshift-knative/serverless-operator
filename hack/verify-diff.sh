#!/usr/bin/env bash

set -euo pipefail

# Generated Files that are allowed to change.
# Changes are still validated against allowed diff patterns below.
readonly -a EXCLUDE_FILES=(
  'olm-catalog/serverless-operator/manifests/serverless-operator.clusterserviceversion.yaml'
  'olm-catalog/serverless-operator-index/Dockerfile'
  'test/images-rekt.yaml'
)

# Directories allowed to change
readonly -a EXCLUDE_DIRS=(
  '.konflux-release/'
)

# Allowed diff patterns inside excluded files.
readonly -a EXCLUDE_PATTERNS=(
  '^[+-].*sha256:'
  '^[+-].*revision: '
  '^[+-].*createdAt: '
  '^[+-].*url: "https://github.com/openshift-knative/.*(\.git)?"'
  '^[+-].*name: serverless-operator-.*-override-snapshot-.*'
)

function debug_log_fail() {
  echo '::debug::Running `git status`'
  git -c color.status=always status

  echo '::debug::Running `git diff`'
  git --no-pager -c color.ui=always diff

  echo '::error::Not all generated files are committed. Run `make generated-files` and commit files.'
  echo '::warning::`make generated-files` needs to be run on GOPATH due to https://github.com/knative/pkg/issues/1287'
}

function is_excluded_file() {
  local file="$1"

  # Exact file matches
  for excluded in "${EXCLUDE_FILES[@]}"; do
    if [[ "$file" == "$excluded" ]]; then
      return 0
    fi
  done

  # Directory prefix matches
  for dir in "${EXCLUDE_DIRS[@]}"; do
    if [[ "$file" == "$dir"* ]]; then
      return 0
    fi
  done

  return 1
}

function validate_excluded_file_diff() {
  local file="$1"

  local diff_lines

  diff_lines=$(
    git --no-pager diff --unified=0 -- "$file" \
      | grep -E '^[+-]' \
      | grep -vE '^(\+\+\+|---|@@)' || true
  )

  # No relevant diff lines
  [[ -z "$diff_lines" ]] && return 0

  while IFS= read -r line; do
    [[ -z "$line" ]] && continue

    local matched=false

    for pattern in "${EXCLUDE_PATTERNS[@]}"; do
      if [[ "$line" =~ $pattern ]]; then
        matched=true
        break
      fi
    done

    if [[ "$matched" == "false" ]]; then
      echo "line '$line' doesn't match any allowed patterns"
      return 1
    fi

  done <<< "$diff_lines"

  return 0
}
# Exit early when there are no changes
git diff --quiet && exit 0

# Collect changed files safely
mapfile -t changed_files < <(git diff --name-only)

# Fail if unexpected files changed
for file in "${changed_files[@]}"; do
  if ! is_excluded_file "$file"; then
    echo "Unexpected modified file: $file"
    debug_log_fail
    exit 33
  fi
done

# Validate allowed diffs in excluded files
for file in "${changed_files[@]}"; do
  if is_excluded_file "$file"; then
    if ! validate_excluded_file_diff "$file"; then
      git --no-pager -c color.ui=always diff -- "$file"
      debug_log_fail
      exit 33
    fi
  fi
done
