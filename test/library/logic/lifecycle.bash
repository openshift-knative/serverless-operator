#!/usr/bin/env bash

include ui/logger.bash
include logic/facts.bash
include infra/ocp/catalogsource.bash
include infra/ocp/namespaces.bash
include infra/ocp/users.bash

function initialize {
  if [[ "${TEARDOWN}" == "on_exit" ]]; then
    logger.debug 'Registering trap for teardown as EXIT'
    trap teardown EXIT
    return 0
  fi
  if [[ "${TEARDOWN}" == "at_start" ]]; then
    teardown
    return 0
  fi
  logger.error "TEARDOWN should only have a one of values: \"on_exit\", \"at_start\", but given: ${TEARDOWN}."
  return 2
}

function teardown {
  logger.warn "Teardown 💀"
  delete_namespaces
  delete_catalog_source
  delete_users
}
