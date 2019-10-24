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
readonly COLOR_GRAY='\e[0;39m'

readonly LOG_LEVEL=${LOG_LEVEL:-INFO}
declare -ar LOG_LEVELS=('DEBUG' 'INFO' 'SUCCESS' 'WARNING' 'ERROR')
declare -Ar LOG_LEVEL_VALUES=( ['DEBUG']=1 ['INFO']=2 ['SUCCESS']=3 ['WARNING']=4 ['ERROR']=5 )

function logger.debug {
  local message
  message="$*"
  if logger.__should-print 'DEBUG'; then
    logger.__log 'DEBUG' "${COLOR_BLUE}" "${message}"
  fi
}

function logger.info {
  local message
  message="$*"
  if logger.__should-print 'INFO'; then
    logger.__log 'INFO' "${COLOR_GREEN}" "${message}"
  fi
}

function logger.success {
  local message
  message="$*"
  if logger.__should-print 'SUCCESS'; then
    logger.__log 'SUCCESS' "${COLOR_LIGHT_GREEN}" "${message}"
  fi
}

function logger.warn {
  local message
  message="$*"
  if logger.__should-print 'WARNING'; then
    logger.__log 'WARNING' "${COLOR_LIGHT_YELLOW}" "${message}"
  fi
}

function logger.error {
  local message
  message="$*"
  if logger.__should-print 'ERROR'; then
    logger.__log 'ERROR' "${COLOR_LIGHT_RED}" "${message}"
  fi
}

function logger.__log {
  local message level now color
  level="$1"
  color="$2"
  message="$3"
  if [[ "${SHOULD_COLOR}" == "true" ]]; then
    now="$(date '+%H:%M:%S.%3N')"
    printf "${color}%7s ${COLOR_CYAN}%s ${color}%s${COLOR_NC}\n" "${level}" "${now}" "${message}" 1>&2
  else
    now="$(date --rfc-3339=ns)"
    printf "%7s %s %s\n" "${level}" "${now}" "${message}" 1>&2
  fi
}

function logger.__check-level {
  local level check_level level_int check_level_int
  level="$1"
  check_level="$2"

  if ! array.contains "$LOG_LEVEL" "${LOG_LEVELS[@]}"; then
    echo "Given invalid log level: ${LOG_LEVEL}, possible values are: ${LOG_LEVELS[*]}" 1>&2
    exit 1
  fi
  
  level_int=${LOG_LEVEL_VALUES[$level]}
  check_level_int=${LOG_LEVEL_VALUES[$check_level]}

  (( level_int >= check_level_int ))
}

function logger.__should-print {
  local level
  level="$1"

  logger.__check-level "$level" "$LOG_LEVEL"
}

logger.info "Actual log level is: ${LOG_LEVEL}. Configure logging by setting LOG_LEVEL env variable."
