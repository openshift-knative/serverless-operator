#!/usr/bin/env bash

if [ -t 1 ]; then 
  IS_TTY=true
else
  IS_TTY=false
fi
readonly IS_TTY

readonly FORCE_COLOR="${FORCE_COLOR:-false}"

SHOULD_COLOR="$(if [[ "${FORCE_COLOR}" == "true" ]] || [[ "${IS_TTY}" == "true" ]]; then echo true; else echo false; fi)"
readonly SHOULD_COLOR

readonly COLOR_NC='\e[0m' # No Color
readonly COLOR_BLUE='\e[0;34m'
readonly COLOR_GREEN='\e[0;32m'
readonly COLOR_LIGHT_GREEN='\e[1;32m'
readonly COLOR_CYAN='\e[0;36m'
readonly COLOR_LIGHT_RED='\e[1;31m'
readonly COLOR_LIGHT_YELLOW='\e[1;33m'

function debugging.setup {
  local debuglog debugdir
  debugdir="${ARTIFACTS:-/tmp}"
  debuglog="${debugdir}/debuglog-$(basename "$0").log"
  logger.info "Debug log (set -x) is written to: ${debuglog}"
  # ref: https://serverfault.com/a/579078
  # Use FD 19 to capture the debug stream caused by "set -x":
  exec 19>> "$debuglog" # Allow appending to the file if exists
  # Tell bash about it  (there's nothing special about 19, its arbitrary)
  export BASH_XTRACEFD=19

  # Register finish of debugging at exit
  trap debugging.finish EXIT
  set -x
}

function debugging.finish {
  # Close the output:
  set +x
  exec 19>&-
}

function logger.debug {
  logger.__log 'DEBUG' "${COLOR_BLUE}" "$*"
}

function logger.info {
  logger.__log 'INFO' "${COLOR_GREEN}" "$*"
}

function logger.success {
  logger.__log 'SUCCESS' "${COLOR_LIGHT_GREEN}" "$*"
}

function logger.warn {
  logger.__log 'WARNING' "${COLOR_LIGHT_YELLOW}" "$*"
}

function logger.error {
  logger.__log 'ERROR' "${COLOR_LIGHT_RED}" "$*"
}

function logger.__log {
  local message level now color ln
  level="$1"
  color="$2"
  message="$3"
  now="$(date '+%H:%M:%S.%3N')"
  ln="${ln:-\n}"
  
  printf "${color}%7s ${COLOR_CYAN}%s ${color}%s${COLOR_NC}${ln}" "${level}" "${now}" "${message}" 1>&2
}

if [[ "${SHOULD_COLOR}" == "false" ]]; then
  function logger.debug {
    echo 'DEBUG' "$(date '+%H:%M:%S.%3N')" "$*"
  }

  function logger.info {
    echo 'INFO' "$(date '+%H:%M:%S.%3N')" "$*"
  }

  function logger.success {
    echo 'SUCCESS' "$(date '+%H:%M:%S.%3N')" "$*"
  }

  function logger.warn {
    echo 'WARNING' "$(date '+%H:%M:%S.%3N')" "$*"
  }

  function logger.error {
    echo 'ERROR' "$(date '+%H:%M:%S.%3N')" "$*"
  }
fi
