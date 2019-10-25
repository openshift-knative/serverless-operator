#!/usr/bin/env bash

declare -a __sources=(vars common ui scaleup namespaces catalogsource servicemesh serverless)

for source in "${__sources[@]}"; do
  # shellcheck disable=SC1091,SC1090
  source "$(dirname "${BASH_SOURCE[0]}")/${source}.bash"
done

unset __sources
