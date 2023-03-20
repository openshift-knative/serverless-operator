#!/usr/bin/env bash

function array.join {
  local IFS="$1"
  shift
  echo "$*"
}

# Loops until duration (car) is exceeded or command (cdr) returns non-zero
function timeout {
  local seconds timeout
  timeout="${1:?Pass a timeout as arg[1]}"
  interval="${interval:-1}"
  seconds=0
  shift
  ln=' ' logger.debug "${*} : Waiting until non-zero (max ${timeout} sec.)"
  while (eval "$*" 2>/dev/null); do
    seconds=$(( seconds + interval ))
    echo -n '.'
    sleep "$interval"
    [[ $seconds -gt $timeout ]] && echo '' \
      && logger.error "Time out of ${timeout} exceeded" \
      && return 71
  done
  [[ $seconds -gt 0 ]] && echo -n ' '
  echo 'done'
  return 0
}

function wait_for_file {
  local file timeout waits
  file="${1:?Pass a filepath as arg[1]}"
  waits="${2:-300}"

  timeout "${waits}" "[[ ! -f '${file}' ]]"
}

function versions.le {
  local v1 v2 cmp
  v1="${1:?Pass a version to check as arg[1]}"
  v2="${2:?Pass a version to check against as arg[2]}"
  cmp="$(echo -e "${v1}\n${v2}" | sort -V | head -n 1)"

  [ "${v1}" = "${cmp}" ]
}

function versions.ge {
  local v1 v2 cmp
  v1="${1:?Pass a version to check as arg[1]}"
  v2="${2:?Pass a version to check against as arg[2]}"
  cmp="$(echo -e "${v1}\n${v2}" | sort -V | tail -n 1)"

  [ "${v1}" = "${cmp}" ]
}

function versions.lt {
  local v1 v2
  v1="${1:?Pass a version to check as arg[1]}"
  v2="${2:?Pass a version to check against as arg[2]}"

  if ! [ "${v1}" = "${v2}" ]; then
    return 1
  fi

  versions.le "${v1}" "${v2}"
}

# Returns the major and minor part of the whole version, joined with a dot.
function versions.major_minor {
  local version=${1:?Pass a full version as arg[1]}
  # shellcheck disable=SC2001
  # Ref: https://regex101.com/r/Po1HA3/1
  echo "${version}" | sed 's/^v\?\([[:digit:]]\+\)\.\([[:digit:]]\+\).*/\1.\2/'
}

# Breaks all image references in the passed YAML file.
function yaml.break_image_references {
  sed -i "s,image: .*,image: TO_BE_REPLACED," "$1"
  sed -i "s,value: gcr.io/knative-releases.*,value: TO_BE_REPLACED," "$1"
}

function should_run {
  local ts
  ts=${1:?Specify test to check}

  if [ -n "$OPENSHIFT_CI" ]; then
    grep -q -e "All" -e "$ts" "${ARTIFACT_DIR}/tests.txt" || return 1
  else
    return 0
  fi
}
