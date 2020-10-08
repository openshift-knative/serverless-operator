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

function wait_for_file {
  local file timeout waits
  file="${1:?Pass a filepath as arg[1]}"
  waits="${2:-300}"
  timeout=$waits

  logger.debug "Waiting for existence of file: ${file}"

  while [ ! -f "${file}" ]; do
    # When the timeout is equal to zero, show an error and leave the loop.
    if [ "${timeout}" == 0 ]; then
      logger.error "Timeout (${waits}s) while waiting for the file ${file}."
      return 78
    fi

    sleep 1

    # Decrease the timeout of one
    ((timeout--))
  done
  return 0
}
