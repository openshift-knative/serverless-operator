#!/usr/bin/env bash

# Removes the `trustBundles` key from contract configmaps.
#
# Usage
# downgrade.sh <namespace> <additional-configmaps>
#   Example:
#   downgrade.sh knative-eventing kafka-source-dispatcher-0 kafka-source-dispatcher-1
#
# namespace: Optional - The namespace in which to update the configmaps. Defaults to "knative-eventing"
# additional-configmaps: Optional - List of additional contract configmaps to patch
#   (in addition to kafka-broker-brokers-triggers, kafka-channel-channels-subscriptions and kafka-sink-sinks)

target_namespace="${1:-"knative-eventing"}"
declare -a contract_configmaps=("kafka-broker-brokers-triggers" "kafka-channel-channels-subscriptions" "kafka-sink-sinks")

for additional_cm in "${@:2}"; do
  contract_configmaps+=("$additional_cm")
done

for cm_name in "${contract_configmaps[@]}"; do
  cmdata=$(kubectl get cm "$cm_name" -n "$target_namespace" -ojson || true)
  if [ -n "$cmdata" ]; then
    new_data=$(echo "$cmdata" | jq -r .binaryData.data | base64 --decode | jq 'del(.trustBundles, .resources[].ingress.enableAutoCreateEventTypes)' -c | base64 -w 0)
    echo "$cmdata" | jq --arg new_data "$new_data" '.binaryData.data = $new_data' | kubectl apply -f -
  fi
done
