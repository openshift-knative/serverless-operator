#!/usr/bin/env bash

function array.join {
  local IFS="$1"
  shift
  echo "$*"
}

# Loops until duration (car) is exceeded or command (cdr) returns non-zero
function timeout {
  local seconds timeout interval
  interval=5
  seconds=0
  timeout=$1
  shift
  while eval $*; do
    seconds=$(( seconds + interval ))
    logger.debug "Execution failed: ${*}. Waiting ${interval} sec ($seconds/${timeout})..."
    sleep $interval
    [[ $seconds -gt $timeout ]] && logger.error "Time out of ${timeout} exceeded" && return 1
  done
  return 0
}

function wait_for_file {
  local file timeout waits
  file="${1:?Pass a filepath as arg[1]}"
  waits="${2:-300}"

  timeout "${waits}" "[[ ! -f '${file}' ]]" || return $?
}

function versions.le {
  local v1 v2 cmp
  v1="${1:?Pass a version to check as arg[1]}"
  v2="${2:?Pass a version to check against as arg[2]}"
  cmp="$(echo -e "${v1}\n${v2}" | sort -V | head -n 1)"

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
