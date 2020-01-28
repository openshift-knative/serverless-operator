#!/usr/bin/env bash

function logger.debug {
  printf "%7s %s %s\n" "DEBUG" "$(date '+%H:%M:%S.%3N')" "$*" 1>&2
}

function logger.info {
  printf "%7s %s %s\n" "INFO" "$(date '+%H:%M:%S.%3N')" "$*" 1>&2
}

function logger.success {
  printf "%7s %s %s\n" "SUCCESS" "$(date '+%H:%M:%S.%3N')" "$*" 1>&2
}

function logger.warn {
  printf "%7s %s %s\n" "WARNING" "$(date '+%H:%M:%S.%3N')" "$*" 1>&2
}

function logger.error {
  printf "%7s %s %s\n" "ERROR" "$(date '+%H:%M:%S.%3N')" "$*" 1>&2
}
