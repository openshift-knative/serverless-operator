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

# Loops until duration (car) is exceeded or command (cdr) returns non-zero
function timeout {
  local timeout
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
