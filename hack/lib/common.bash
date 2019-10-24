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
    if [[ "${LOG_LEVEL}" != 'DEBUG' ]]; then
      echo -n '.'
    fi
    sleep $interval
    [[ $seconds -gt $timeout ]] && logger.error "Timed out of ${timeout} exceeded" && return 1
  done
  if [[ "${LOG_LEVEL}" != 'DEBUG' ]] && [[ "$seconds" != '0' ]]; then
    echo ''
  fi
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
