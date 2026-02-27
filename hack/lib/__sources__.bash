#!/usr/bin/env bash

declare -a __sources=(metadata vars images common ui scaleup namespaces serverless catalog olmv0_catalog olmv1_catalog tracing mesh certmanager strimzi keda tracing clusterlogging testselect)

for source in "${__sources[@]}"; do
  # shellcheck disable=SC1091,SC1090
  source "$(dirname "${BASH_SOURCE[0]}")/${source}.bash"
done

unset __sources
