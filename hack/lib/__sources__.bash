#!/usr/bin/env bash

declare -a __sources=(metadata vars common ui scaleup namespaces catalogsource serverless tracing mesh strimzi)

for source in "${__sources[@]}"; do
  # shellcheck disable=SC1091,SC1090
  source "$(dirname "${BASH_SOURCE[0]}")/${source}.bash"
done

unset __sources
