#!/usr/bin/env bash

function install_strimzi_operator {
  logger.info "Installing Strimzi Kafka operator"
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
  logger.info "Applying Strimzi Cluster file"
  cat <<-EOF | oc apply -f -
    apiVersion: kafka.strimzi.io/v1beta2
    kind: Kafka
    metadata:
      name: my-cluster
      namespace: kafka
    spec:
      kafka:
        version: 3.6.1
        replicas: 3
        listeners:
          # PLAINTEXT
          - name: plain
            port: 9092
            type: internal
            tls: false
          # SSL
          - name: tls
            port: 9093
            type: internal
            tls: true
            authentication:
              type: tls
          # protocol=SASL_SSL
          # sasl.mechanism=SCRAM-SHA-512
          - name: saslssl
            port: 9094
            type: internal
            tls: true
            authentication:
              type: scram-sha-512
          # protocol=SASL_PLAINTEXT
          # sasl.mechanism=SCRAM-SHA-512
          - name: saslplain
            port: 9095
            type: internal
            tls: false
            authentication:
              type: scram-sha-512
          # TLS no auth
          - name: tlsnoauth
            port: 9096
            type: internal
            tls: true
        authorization:
          superUsers:
            - ANONYMOUS
          type: simple
        config:
          offsets.topic.replication.factor: 3
          transaction.state.log.replication.factor: 3
          transaction.state.log.min.isr: 2
          inter.broker.protocol.version: "3.6"
          auto.create.topics.enable: "false"
        storage:
          type: jbod
          volumes:
          - id: 0
            type: persistent-claim
            size: 100Gi
            deleteClaim: false
        resources:
          requests:
            memory: 2Gi
            cpu: "300m"
          limits:
            memory: 4Gi
            cpu: "4"
      zookeeper:
        replicas: 3
        storage:
          type: persistent-claim
          size: 100Gi
          deleteClaim: false
        resources:
          requests:
            memory: 500Mi
            cpu: "300m"
          limits:
            memory: 2Gi
            cpu: "2"
      entityOperator:
        topicOperator: {}
        userOperator: {}
EOF

  logger.info "Waiting for Strimzi cluster to become ready"
  oc wait kafka --all --timeout=-1s --for=condition=Ready -n kafka
}

function install_strimzi_users {
  logger.info "Applying Strimzi TLS Admin user"
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
      - resource:
          type: group
          name: "*"
        operation: Delete
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
      # Required ACL rule to be able to delete topics
      - resource:
          type: topic
          name: "*"
        operation: Delete
        host: "*"
EOF

  logger.info "Applying Strimzi SASL Admin User"
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
      - resource:
          type: group
          name: "*"
        operation: Delete
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
      # Required ACL rule to be able to delete topics
      - resource:
          type: topic
          name: "*"
        operation: Delete
        host: "*"
EOF

  logger.info "Applying Strimzi SASL Restricted User"
  cat <<-EOF | oc apply -f -
apiVersion: kafka.strimzi.io/v1beta2
kind: KafkaUser
metadata:
  name: my-restricted-sasl-user
  namespace: kafka
  labels:
    strimzi.io/cluster: my-cluster
spec:
  authentication:
    type: scram-sha-512
  authorization:
    type: simple
    acls:
      # Example ACL rules for Broker with names following knative default brokers.topic.template
      - resource:
          type: topic
          name: knative-broker-
          patternType: prefix
        operations:
          - Create
          - Describe
          - Read
          - Write
          - Delete
        host: "*"
      # Example ACL rules for Consumer Group ID following knative default triggers.consumergroup.template
      - resource:
          type: group
          name: knative-trigger-
          patternType: prefix
        operations:
          - Read
        host: "*"
EOF

  logger.info "Waiting for Strimzi admin users to become ready"
  oc wait kafkauser --all --timeout=-1s --for=condition=Ready -n kafka

  logger.info "Deleting existing Kafka user secrets"

  oc delete secret -n default my-tls-secret --ignore-not-found
  oc delete secret -n default my-sasl-secret --ignore-not-found
  oc delete secret -n "${EVENTING_NAMESPACE}" strimzi-tls-secret --ignore-not-found
  oc delete secret -n "${EVENTING_NAMESPACE}" strimzi-sasl-secret --ignore-not-found
  oc delete secret -n "${EVENTING_NAMESPACE}" strimzi-sasl-secret-legacy --ignore-not-found
  oc delete secret -n "${EVENTING_NAMESPACE}" strimzi-tls-secret-legacy --ignore-not-found

  logger.info "Creating a Secret, containing TLS from Strimzi"
  STRIMZI_CRT=$(oc -n kafka get secret my-cluster-cluster-ca-cert --template='{{index .data "ca.crt"}}' | base64 --decode )
  TLSUSER_CRT=$(oc -n kafka get secret my-tls-user --template='{{index .data "user.crt"}}' | base64 --decode )
  TLSUSER_KEY=$(oc -n kafka get secret my-tls-user --template='{{index .data "user.key"}}' | base64 --decode )

  oc create secret --namespace default generic my-tls-secret \
      --from-literal=ca.crt="$STRIMZI_CRT" \
      --from-literal=user.crt="$TLSUSER_CRT" \
      --from-literal=user.key="$TLSUSER_KEY"

  logger.info "Creating a Secret, containing SASL from Strimzi"
  SASL_PASSWD=$(oc -n kafka get secret my-sasl-user --template='{{index .data "password"}}' | base64 --decode )
  oc create secret --namespace default generic my-sasl-secret \
      --from-literal=ca.crt="$STRIMZI_CRT" \
      --from-literal=password="$SASL_PASSWD" \
      --from-literal=saslType="SCRAM-SHA-512" \
      --from-literal=user="my-sasl-user"

  oc create secret --namespace "${EVENTING_NAMESPACE}" generic strimzi-tls-secret \
    --from-literal=ca.crt="$STRIMZI_CRT" \
    --from-literal=user.crt="$TLSUSER_CRT" \
    --from-literal=user.key="$TLSUSER_KEY" \
    --from-literal=protocol="SSL" \
    --dry-run=client -o yaml | oc apply -n "${EVENTING_NAMESPACE}" -f -

  oc create secret --namespace "${EVENTING_NAMESPACE}" generic strimzi-sasl-secret \
    --from-literal=ca.crt="$STRIMZI_CRT" \
    --from-literal=password="$SASL_PASSWD" \
    --from-literal=user="my-sasl-user" \
    --from-literal=protocol="SASL_SSL" \
    --from-literal=sasl.mechanism="SCRAM-SHA-512" \
    --from-literal=saslType="SCRAM-SHA-512" \
    --dry-run=client -o yaml | oc apply -n "${EVENTING_NAMESPACE}" -f -

  oc create secret --namespace "${EVENTING_NAMESPACE}" generic strimzi-sasl-secret-legacy \
    --from-literal=ca.crt="$STRIMZI_CRT" \
    --from-literal=password="$SASL_PASSWD" \
    --from-literal=user="my-sasl-user" \
    --from-literal=saslType="SCRAM-SHA-512" \
    --dry-run=client -o yaml | oc apply -n "${EVENTING_NAMESPACE}" -f -

  oc create secret --namespace "${EVENTING_NAMESPACE}" generic strimzi-sasl-plain-secret \
    --from-literal=password="$SASL_PASSWD" \
    --from-literal=user="my-sasl-user" \
    --from-literal=protocol="SASL_PLAINTEXT" \
    --from-literal=sasl.mechanism="SCRAM-SHA-512" \
    --from-literal=saslType="SCRAM-SHA-512" \
    --dry-run=client -o yaml | oc apply -n "${EVENTING_NAMESPACE}" -f -

  oc create secret --namespace "${EVENTING_NAMESPACE}" generic strimzi-sasl-plain-secret-legacy \
    --from-literal=password="$SASL_PASSWD" \
    --from-literal=username="my-sasl-user" \
    --from-literal=saslType="SCRAM-SHA-512" \
    --dry-run=client -o yaml | oc apply -n "${EVENTING_NAMESPACE}" -f -
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
  securityContext:
    runAsNonRoot: true
    seccompProfile:
      type: RuntimeDefault
  containers:
    - env:
      - name: KAFKA_CLUSTERS_0_BOOTSTRAPSERVERS
        value: my-cluster-kafka-bootstrap.kafka.svc:9092
      - name: KAFKA_CLUSTERS_0_NAME
        value: my-cluster
      image: quay.io/openshift-knative/kafka-ui:0.1.0
      name: user-container
      securityContext:
        runAsUser: 1000
        allowPrivilegeEscalation: false
        capabilities:
          drop:
          - ALL
EOF

  oc -n kafka expose pod kafka-ui --port=8080
  oc -n kafka expose service kafka-ui

  timeout 600 "[[ \$(oc get -n kafka route.route.openshift.io/kafka-ui -ojsonpath='{.status.ingress[0].host}') == '' ]]" || return 2

  logger.success "Kafka UI URL: $(oc get -n kafka route.route.openshift.io/kafka-ui  -ojsonpath='{.status.ingress[0].host}')"
}

function install_strimzi {
  logger.info "Strimzi install"
  oc create namespace "${EVENTING_NAMESPACE}" --dry-run=client -o yaml | oc apply -f -
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
  logger.info "Deleting Kafka user secrets"
  oc delete secret -n default my-tls-secret
  oc delete secret -n default my-sasl-secret

  logger.info "Deleting Strimzi users"
  oc -n kafka delete kafkauser.kafka.strimzi.io my-sasl-user
  oc -n kafka delete kafkauser.kafka.strimzi.io my-tls-user

  logger.info "Waiting for Strimzi users to get deleted"
  timeout 600 "[[ \$(oc get kafkausers -n kafka -o jsonpath='{.items}') != '[]' ]]" || return 2
}

function delete_strimzi_cluster {
  logger.info "Deleting Strimzi cluster"
  oc delete kafka -n kafka my-cluster

  logger.info "Waiting for Strimzi cluster to get deleted"
  timeout 600 "[[ \$(oc get kafkas -n kafka -o jsonpath='{.items}') != '[]' ]]" || return 2
}

function delete_strimzi_operator {
  logger.info "Deleting Strimzi Kafka operator"

  curl -L "https://github.com/strimzi/strimzi-kafka-operator/releases/download/${STRIMZI_VERSION}/strimzi-cluster-operator-${STRIMZI_VERSION}.yaml" \
  | sed 's/namespace: .*/namespace: kafka/' \
  | oc -n kafka delete -f -

  oc -n kafka delete --selector strimzi.io/crd-install=true -f "https://github.com/strimzi/strimzi-kafka-operator/releases/download/${STRIMZI_VERSION}/strimzi-cluster-operator-${STRIMZI_VERSION}.yaml"

  oc delete namespace kafka
}

function uninstall_strimzi {
  logger.info "Strimzi uninstall"
  delete_kafka_ui
  delete_strimzi_users
  delete_strimzi_cluster
  delete_strimzi_operator
}
