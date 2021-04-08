#!/usr/bin/env bash

function install_strimzi {
  strimzi_version=`curl https://github.com/strimzi/strimzi-kafka-operator/releases/latest |  awk -F 'tag/' '{print $2}' | awk -F '"' '{print $1}' 2>/dev/null`
  header "Strimzi install"
  oc create namespace kafka
  curl -L "https://github.com/strimzi/strimzi-kafka-operator/releases/download/${strimzi_version}/strimzi-cluster-operator-${strimzi_version}.yaml" \
  | sed 's/namespace: .*/namespace: kafka/' \
  | oc -n kafka create -f -

  # Wait for the CRD we need to actually be active
  oc wait crd --timeout=-1s kafkas.kafka.strimzi.io --for=condition=Established

  header "Applying Strimzi Cluster file"
  oc -n kafka apply -f "https://raw.githubusercontent.com/strimzi/strimzi-kafka-operator/${strimzi_version}/examples/kafka/kafka-persistent.yaml"

  header "Waiting for Strimzi to become ready"
  oc wait kafka --all --timeout=-1s --for=condition=Ready -n kafka
}

function uninstall_strimzi {
  strimzi_version=`curl https://github.com/strimzi/strimzi-kafka-operator/releases/latest |  awk -F 'tag/' '{print $2}' | awk -F '"' '{print $1}' 2>/dev/null`

  header "Deleting Kafka instance"
  oc -n kafka delete -f "https://raw.githubusercontent.com/strimzi/strimzi-kafka-operator/${strimzi_version}/examples/kafka/kafka-persistent.yaml"

  header "Waiting for Kafka to get deleted"
  timeout 600 "[[ \$(oc get kafkas -n kafka -o jsonpath='{.items}') != '[]' ]]" || return 2

  header "Deleting Strimzi Cluster file"
  curl -L "https://github.com/strimzi/strimzi-kafka-operator/releases/download/${strimzi_version}/strimzi-cluster-operator-${strimzi_version}.yaml" \
  | sed 's/namespace: .*/namespace: kafka/' \
  | oc -n kafka delete -f -

  oc -n kafka delete --selector strimzi.io/crd-install=true -f "https://github.com/strimzi/strimzi-kafka-operator/releases/download/${strimzi_version}/strimzi-cluster-operator-${strimzi_version}.yaml"

  oc delete namespace kafka
}
