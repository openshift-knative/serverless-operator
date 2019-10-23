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
readonly COLOR_WHITE='\e[1;37m'
readonly COLOR_BLACK='\e[0;30m'
readonly COLOR_BLUE='\e[0;34m'
readonly COLOR_LIGHT_BLUE='\e[1;34m'
readonly COLOR_GREEN='\e[0;32m'
readonly COLOR_LIGHT_GREEN='\e[1;32m'
readonly COLOR_CYAN='\e[0;36m'
readonly COLOR_LIGHT_CYAN='\e[1;36m'
readonly COLOR_RED='\e[0;31m'
readonly COLOR_LIGHT_RED='\e[1;31m'
readonly COLOR_PURPLE='\e[0;35m'
readonly COLOR_LIGHT_PURPLE='\e[1;35m'
readonly COLOR_BROWN='\e[0;33m'
readonly COLOR_YELLOW='\e[1;33m'
readonly COLOR_GRAY='\e[0;30m'
readonly COLOR_LIGHT_GRAY='\e[0;37m'

readonly LOG_LEVEL=${LOG_LEVEL:-INFO}

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
  if logger.__should-print 'INFO'; then
    logger.__log 'INFO' "${COLOR_LIGHT_GREEN}" "${message}"
  fi
}

function logger.warn {
  local message
  message="$*"
  if logger.__should-print 'WARN'; then
    logger.__log 'WARN' "${COLOR_YELLOW}" "${message}"
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
    printf "${color}%5s ${COLOR_CYAN}%s ${color}%s${COLOR_NC}\n" "${level}" "${now}" "${message}" 1>&2
  else
    now="$(date --rfc-3339=ns)"
    printf "%5s %s %s\n" "${level}" "${now}" "${message}" 1>&2
  fi
}

function logger.__should-print {
  local level
  level="$1"
  local log_levels
  log_levels=('DEBUG' 'INFO' 'WARN' 'ERROR')
  declare -A log_level_values=( ['DEBUG']=1 ['INFO']=2 ['WARN']=3 ['ERROR']=4 )

  if ! array.contains "$LOG_LEVEL" "${log_levels[@]}"; then
    echo "Given invalid log level: ${LOG_LEVEL}, possible values are: ${log_levels[*]}" 1>&2
    exit 1
  fi
  local int_level
  int_level=${log_level_values[$level]}
  local int_displaying
  int_displaying=${log_level_values[$LOG_LEVEL]}
  (( int_level >= int_displaying ))
}
