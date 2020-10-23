#!/usr/bin/env bash

function array.contains {
  local e match="$1"
  shift
  for e; do [[ "$e" == "$match" ]] && return 0; done
  return 1
}

function array.join {
  local IFS="$1"
  shift
  echo "$*"
}

function resolve_hostname {
  local ip
  ip="$(LANG=C host -t a "${1}" | grep 'has address' | head -n 1 | awk '{print $4}')"
  if [ "${ip}" != "" ]; then
    echo "${ip}"
  fi
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
}

function wait_until_pods_running {
  logger.error "Upstream function wait_until_pods_running doesn't work well \
with strict Bash rules that should be used while scripting, and shouldn't be \
used in OpenShift Serverless. Please use wait_until_labelled_pods_are_ready \
instead"
  exit 88
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

# Waits until the given hostname resolves via DNS
# Parameters: $1 - hostname
function wait_until_hostname_resolves() {
  logger.info "Waiting until hostname $1 resolves via DNS"
  for _ in {1..150}; do  # timeout after 15 minutes
    local ip
    ip="$(resolve_hostname "$1")"
    if [[ "$ip" != "" ]]; then
      echo ''
      logger.info "Resolved as ${ip}"
      return 0
    fi
    echo -n "."
    sleep 6
  done
  echo -e "\n\n"
  logger.error "Timeout waiting for hostname $1 to resolve via DNS"
  return 1
}
