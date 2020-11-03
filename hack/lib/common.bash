#!/usr/bin/env bash

function array.join {
  local IFS="$1"
  shift
  echo "$*"
}

# Waits until labelled pods reports ready. It should be used erroneous, upstream,
# wait_until_pods_running func.
function wait_until_labelled_pods_are_ready {
  local label ns
  label="${1:?Pass a label as arg[1]}"
  ns="${2:?Pass a namespace as arg[2]}"

  # Wait for some pods to sprung
  timeout 300 "[[ \$(oc get pods -l ${label} -n ${ns} -o name | wc -l) == '0' ]]"
  # Wait until they are ready to receive communications
  timeout 300 "[[ \$(oc get pods -l ${label} -n ${ns} -o \
'jsonpath={..status.conditions[?(@.type==\"Ready\")].status}') != 'True' ]]"
  return 0
}

function wait_until_labelled_pods_are_ready {
  local label ns
  label="${1:?Pass a label as arg[1]}"
  ns="${2:?Pass a namespace as arg[2]}"
  timeout 300 "[[ \$(oc get pods -l ${label} -n ${ns} -o \
'jsonpath={..status.conditions[?(@.type==\"Ready\")].status}') != 'True' ]]"
}

# Loops until duration (car) is exceeded or command (cdr) returns non-zero
function timeout {
  local seconds timeout
  interval="${interval:-2}"
  seconds=0
  timeout=${1:?Pass timeout as arg[1]}
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

function versions.lt {
  local v1 v2
  v1="${1:?Pass a version to check as arg[1]}"
  v2="${2:?Pass a version to check against as arg[2]}"

  if ! [ "${v1}" = "${v2}" ]; then
    return 1
  fi

  versions.le "${v1}" "${v2}"
}
