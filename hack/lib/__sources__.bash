#!/usr/bin/env bash

# Temp replace until build-image is updated
if [ -n "$OPENSHIFT_CI" ]; then
  export YQ_TEMP
  YQ_TEMP=$(mktemp -d)
  export PATH="$YQ_TEMP:$PATH:$YQ_TEMP"

  wget https://github.com/mikefarah/yq/releases/download/v4.35.2/yq_linux_amd64 -O "$YQ_TEMP/yq"
  chmod +x "$YQ_TEMP/yq"

  yq --version
fi

declare -a __sources=(metadata vars common ui scaleup namespaces catalogsource serverless tracing mesh certmanager strimzi keda tracing clusterlogging testselect)

for source in "${__sources[@]}"; do
  # shellcheck disable=SC1091,SC1090
  source "$(dirname "${BASH_SOURCE[0]}")/${source}.bash"
done

unset __sources
