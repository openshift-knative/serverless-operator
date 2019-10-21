#!/usr/bin/env bash

include ui/logger.bash
include logic/facts.bash

function run_e2e_tests {
  logger.info "Running tests"
  go test -v -tags=e2e -count=1 -timeout=10m -parallel=1 ./test/e2e \
    --kubeconfig "${KUBECONFIG},$(pwd)/user1.kubeconfig,$(pwd)/user2.kubeconfig" \
    && logger.success 'Tests has passed' && return 0 \
    || logger.error 'Tests have failures!' \
    && return 1
}
