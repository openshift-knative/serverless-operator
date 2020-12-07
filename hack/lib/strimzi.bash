#!/usr/bin/env bash

function install_strimzi {
  strimzi_version=`curl https://github.com/strimzi/strimzi-kafka-operator/releases/latest |  awk -F 'tag/' '{print $2}' | awk -F '"' '{print $1}' 2>/dev/null`
  header "Strimzi install"
  oc create namespace kafka
  oc -n kafka apply --selector strimzi.io/crd-install=true -f "https://github.com/strimzi/strimzi-kafka-operator/releases/download/${strimzi_version}/strimzi-cluster-operator-${strimzi_version}.yaml"
  curl -L "https://github.com/strimzi/strimzi-kafka-operator/releases/download/${strimzi_version}/strimzi-cluster-operator-${strimzi_version}.yaml" \
  | sed 's/namespace: .*/namespace: kafka/' \
  | oc -n kafka apply -f -

  # Wait for the CRD we need to actually be active
  oc wait crd --timeout=-1s kafkas.kafka.strimzi.io --for=condition=Established

  header "Applying Strimzi Cluster file"
  cat <<-EOF | oc apply -f -
    apiVersion: kafka.strimzi.io/v1beta1
    kind: Kafka
    metadata:
      name: my-cluster
      namespace: kafka
    spec:
      kafka:
        version: 2.6.0
        replicas: 3
        listeners:
          - name: plain
            port: 9092
            type: internal
            tls: false
          - name: tls
            port: 9093
            type: internal
            tls: true
            authentication:
              type: tls
          - name: sasl
            port: 9094
            type: internal
            tls: true
            authentication:
              type: scram-sha-512
        config:
          offsets.topic.replication.factor: 3
          transaction.state.log.replication.factor: 3
          transaction.state.log.min.isr: 2
          log.message.format.version: "2.6"
        storage:
          type: jbod
          volumes:
          - id: 0
            type: persistent-claim
            size: 100Gi
            deleteClaim: false
      zookeeper:
        replicas: 3
        storage:
          type: persistent-claim
          size: 100Gi
          deleteClaim: false
      entityOperator:
        topicOperator: {}
        userOperator: {}
EOF

  header "Waiting for Strimzi to become ready"
  oc wait kafka --all --timeout=-1s --for=condition=Ready -n kafka

  header "Applying Strimzi TLS Admin user"
  cat <<-EOF | oc apply -f -
apiVersion: kafka.strimzi.io/v1beta1
kind: KafkaUser
metadata:
  name: my-tls-user
  namespace: kafka
  labels:
    strimzi.io/cluster: my-cluster
spec:
  authentication:
    type: tls
EOF

  header "Applying Strimzi SASL Admin User"
  cat <<-EOF | oc apply -f -
apiVersion: kafka.strimzi.io/v1beta1
kind: KafkaUser
metadata:
  name: my-sasl-user
  namespace: kafka
  labels:
    strimzi.io/cluster: my-cluster
spec:
  authentication:
    type: scram-sha-512
EOF

  header "Waiting for Strimzi admin users to become ready"
  oc wait kafkauser --all --timeout=-1s --for=condition=Ready -n kafka

}

function uninstall_strimzi {
  strimzi_version=`curl https://github.com/strimzi/strimzi-kafka-operator/releases/latest |  awk -F 'tag/' '{print $2}' | awk -F '"' '{print $1}' 2>/dev/null`

  header "Deleting Strimzi users"
  oc -n kafka delete kafkauser.kafka.strimzi.io my-sasl-user
  oc -n kafka delete kafkauser.kafka.strimzi.io my-tls-user

  header "Waiting for Kafka users to get deleted"
  timeout 600 "[[ \$(oc get kafkausers -n kafka -o jsonpath='{.items}') != '[]' ]]" || return 2


  header "Deleting Kafka instance"
  oc delete kafka -n kafka my-cluster

  header "Waiting for Kafka to get deleted"
  timeout 600 "[[ \$(oc get kafkas -n kafka -o jsonpath='{.items}') != '[]' ]]" || return 2

  header "Deleting Strimzi Cluster file"
  curl -L "https://github.com/strimzi/strimzi-kafka-operator/releases/download/${strimzi_version}/strimzi-cluster-operator-${strimzi_version}.yaml" \
  | sed 's/namespace: .*/namespace: kafka/' \
  | oc -n kafka delete -f -

  oc -n kafka delete --selector strimzi.io/crd-install=true -f "https://github.com/strimzi/strimzi-kafka-operator/releases/download/${strimzi_version}/strimzi-cluster-operator-${strimzi_version}.yaml"

  oc delete namespace kafka
}
