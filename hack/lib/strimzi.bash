#!/usr/bin/env bash

function install_strimzi_operator {
  header "Installing Strimzi Kafka operator"
  if ! oc get ns kafka &>/dev/null; then
    oc create namespace kafka
  fi
  oc -n kafka apply --selector strimzi.io/crd-install=true -f "https://github.com/strimzi/strimzi-kafka-operator/releases/download/${STRIMZI_VERSION}/strimzi-cluster-operator-${STRIMZI_VERSION}.yaml"
  curl -L "https://github.com/strimzi/strimzi-kafka-operator/releases/download/${STRIMZI_VERSION}/strimzi-cluster-operator-${STRIMZI_VERSION}.yaml" \
  | sed 's/namespace: .*/namespace: kafka/' \
  | oc -n kafka apply -f -

  # Wait for the CRD we need to actually be active
  oc wait crd --timeout=-1s kafkas.kafka.strimzi.io --for=condition=Established
}

function install_strimzi_cluster {
  header "Applying Strimzi Cluster file"
  cat <<-EOF | oc apply -f -
    apiVersion: kafka.strimzi.io/v1beta2
    kind: Kafka
    metadata:
      name: my-cluster
      namespace: kafka
    spec:
      kafka:
        version: 3.0.0
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
        authorization:
          superUsers:
            - ANONYMOUS
          type: simple
        config:
          offsets.topic.replication.factor: 3
          transaction.state.log.replication.factor: 3
          transaction.state.log.min.isr: 2
          inter.broker.protocol.version: "3.0"
          log.message.format.version: "3.0"
          auto.create.topics.enable: "false"
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

  header "Waiting for Strimzi cluster to become ready"
  oc wait kafka --all --timeout=-1s --for=condition=Ready -n kafka
}

function install_strimzi_users {
  header "Applying Strimzi TLS Admin user"
  cat <<-EOF | oc apply -f -
apiVersion: kafka.strimzi.io/v1beta2
kind: KafkaUser
metadata:
  name: my-tls-user
  namespace: kafka
  labels:
    strimzi.io/cluster: my-cluster
spec:
  authentication:
    type: tls
  authorization:
    type: simple
    acls:
      # Example ACL rules for consuming from a topic.
      - resource:
          type: topic
          name: "*"
        operation: Read
        host: "*"
      - resource:
          type: topic
          name: "*"
        operation: Describe
        host: "*"
      - resource:
          type: group
          name: "*"
        operation: Read
        host: "*"
      # Example ACL rules for producing to a topic.
      - resource:
          type: topic
          name: "*"
        operation: Write
        host: "*"
      - resource:
          type: topic
          name: "*"
        operation: Create
        host: "*"
      - resource:
          type: topic
          name: "*"
        operation: Describe
        host: "*"
EOF

  header "Applying Strimzi SASL Admin User"
  cat <<-EOF | oc apply -f -
apiVersion: kafka.strimzi.io/v1beta2
kind: KafkaUser
metadata:
  name: my-sasl-user
  namespace: kafka
  labels:
    strimzi.io/cluster: my-cluster
spec:
  authentication:
    type: scram-sha-512
  authorization:
    type: simple
    acls:
      # Example ACL rules for consuming from knative-messaging-kafka using consumer group my-group
      - resource:
          type: topic
          name: "*"
        operation: Read
        host: "*"
      - resource:
          type: topic
          name: "*"
        operation: Describe
        host: "*"
      - resource:
          type: group
          name: "*"
        operation: Read
        host: "*"
      # Example ACL rules for producing to topic knative-messaging-kafka
      - resource:
          type: topic
          name: "*"
        operation: Write
        host: "*"
      - resource:
          type: topic
          name: "*"
        operation: Create
        host: "*"
      - resource:
          type: topic
          name: "*"
        operation: Describe
        host: "*"
EOF

  header "Waiting for Strimzi admin users to become ready"
  oc wait kafkauser --all --timeout=-1s --for=condition=Ready -n kafka

  header "Deleting existing Kafka user secrets"

  if oc get secret my-tls-secret -n default >/dev/null 2>&1
  then
    oc delete secret -n default my-tls-secret
  fi

  if oc get secret my-sasl-secret -n default >/dev/null 2>&1
  then
    oc delete secret -n default my-sasl-secret
  fi

  header "Creating a Secret, containing TLS from Strimzi"
  STRIMZI_CRT=$(oc -n kafka get secret my-cluster-cluster-ca-cert --template='{{index .data "ca.crt"}}' | base64 --decode )
  TLSUSER_CRT=$(oc -n kafka get secret my-tls-user --template='{{index .data "user.crt"}}' | base64 --decode )
  TLSUSER_KEY=$(oc -n kafka get secret my-tls-user --template='{{index .data "user.key"}}' | base64 --decode )

  oc create secret --namespace default generic my-tls-secret \
      --from-literal=ca.crt="$STRIMZI_CRT" \
      --from-literal=user.crt="$TLSUSER_CRT" \
      --from-literal=user.key="$TLSUSER_KEY"

  header "Creating a Secret, containing SASL from Strimzi"
  SASL_PASSWD=$(oc -n kafka get secret my-sasl-user --template='{{index .data "password"}}' | base64 --decode )
  oc create secret --namespace default generic my-sasl-secret \
      --from-literal=ca.crt="$STRIMZI_CRT" \
      --from-literal=password="$SASL_PASSWD" \
      --from-literal=saslType="SCRAM-SHA-512" \
      --from-literal=user="my-sasl-user"
}

function install_kafka_ui {
  logger.info "Installing Kafka UI"
  cat <<-EOF | oc apply -f -
---
apiVersion: v1
kind: Pod
metadata:
  name: kafka-ui
  namespace: kafka
  labels:
    app: kafka-ui
spec:
  containers:
    - env:
      - name: KAFKA_CLUSTERS_0_BOOTSTRAPSERVERS
        value: my-cluster-kafka-bootstrap.kafka.svc:9092
      - name: KAFKA_CLUSTERS_0_NAME
        value: my-cluster
      image: quay.io/openshift-knative/kafka-ui:0.1.0
      name: user-container
EOF

  oc -n kafka expose pod kafka-ui --port=8080
  oc -n kafka expose service kafka-ui

  timeout 600 "[[ \$(oc get -n kafka route.route.openshift.io/kafka-ui -ojsonpath='{.status.ingress[0].host}') == '' ]]" || return 2

  logger.success "Kafka UI URL: $(oc get -n kafka route.route.openshift.io/kafka-ui  -ojsonpath='{.status.ingress[0].host}')"
}

function install_strimzi {
  header "Strimzi install"
  install_strimzi_operator
  install_strimzi_cluster
  install_strimzi_users
  install_kafka_ui
}

function delete_kafka_ui {
  logger.info 'Deleting Kafka UI'
  oc delete -n kafka route.route.openshift.io/kafka-ui
  oc delete -n kafka service/kafka-ui
  oc delete -n kafka pod/kafka-ui
}

function delete_strimzi_users {
  header "Deleting Kafka user secrets"
  oc delete secret -n default my-tls-secret
  oc delete secret -n default my-sasl-secret

  header "Deleting Strimzi users"
  oc -n kafka delete kafkauser.kafka.strimzi.io my-sasl-user
  oc -n kafka delete kafkauser.kafka.strimzi.io my-tls-user

  header "Waiting for Strimzi users to get deleted"
  timeout 600 "[[ \$(oc get kafkausers -n kafka -o jsonpath='{.items}') != '[]' ]]" || return 2
}

function delete_strimzi_cluster {
  header "Deleting Strimzi cluster"
  oc delete kafka -n kafka my-cluster

  header "Waiting for Strimzi cluster to get deleted"
  timeout 600 "[[ \$(oc get kafkas -n kafka -o jsonpath='{.items}') != '[]' ]]" || return 2
}

function delete_strimzi_operator {
  header "Deleting Strimzi Kafka operator"

  curl -L "https://github.com/strimzi/strimzi-kafka-operator/releases/download/${STRIMZI_VERSION}/strimzi-cluster-operator-${STRIMZI_VERSION}.yaml" \
  | sed 's/namespace: .*/namespace: kafka/' \
  | oc -n kafka delete -f -

  oc -n kafka delete --selector strimzi.io/crd-install=true -f "https://github.com/strimzi/strimzi-kafka-operator/releases/download/${STRIMZI_VERSION}/strimzi-cluster-operator-${STRIMZI_VERSION}.yaml"

  oc delete namespace kafka
}

function uninstall_strimzi {
  header "Strimzi uninstall"
  delete_kafka_ui
  delete_strimzi_users
  delete_strimzi_cluster
  delete_strimzi_operator
}
