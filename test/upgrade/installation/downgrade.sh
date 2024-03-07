#!/usr/bin/env bash

for cm_name in "$@"; do
  cmdata=$(kubectl get cm "$cm_name" -n knative-eventing -ojson)
  if [ ! -z "$cmdata" ]; then
    new_data=$(echo $cmdata | jq -r .binaryData.data | base64 --decode | jq 'del(.trustBundles)' -c | base64 -w 0)
    echo $cmdata | jq --arg new_data $new_data '.binaryData.data = $new_data' | kubectl apply -f -
  fi
done
