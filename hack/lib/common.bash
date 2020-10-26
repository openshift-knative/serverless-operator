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
