#!/usr/bin/env bash

export BUILD_NUMBER=${BUILD_NUMBER:-$(head -c 128 < /dev/urandom | base64 | fold -w 8 | head -n 1)}

if [[ -n "${ARTIFACT_DIR:-}" ]]; then
  ARTIFACTS="${ARTIFACT_DIR}/build-${BUILD_NUMBER}"
  export ARTIFACTS
  mkdir -p "${ARTIFACTS}"
fi

# shellcheck disable=SC1091,SC1090
source "$(dirname "${BASH_SOURCE[0]}")/../../vendor/knative.dev/hack/e2e-tests.sh"

export STRIMZI_VERSION=0.36.1

# Adjust these when upgrading the knative versions.
export KNATIVE_SERVING_VERSION="${KNATIVE_SERVING_VERSION:-$(metadata.get dependencies.serving)}"
export KNATIVE_SERVING_VERSION_PREVIOUS="${KNATIVE_SERVING_VERSION_PREVIOUS:-$(metadata.get dependencies.previous.serving)}"
export KNATIVE_EVENTING_VERSION="${KNATIVE_EVENTING_VERSION:-$(metadata.get dependencies.eventing)}"
export KNATIVE_EVENTING_VERSION_PREVIOUS="${KNATIVE_EVENTING_VERSION_PREVIOUS:-$(metadata.get dependencies.previous.eventing)}"
export KNATIVE_EVENTING_KAFKA_BROKER_VERSION="${KNATIVE_EVENTING_KAFKA_BROKER_VERSION:-$(metadata.get dependencies.eventing_kafka_broker)}"
export KNATIVE_EVENTING_ISTIO_VERSION="${KNATIVE_EVENTING_ISTIO_VERSION:-$(metadata.get dependencies.eventing_istio)}"
export KNATIVE_EVENTING_KAFKA_BROKER_VERSION_PREVIOUS="${KNATIVE_EVENTING_KAFKA_BROKER_VERSION_PREVIOUS:-$(metadata.get dependencies.previous.eventing_kafka_broker)}"

CURRENT_VERSION="$(metadata.get project.version)"
PREVIOUS_VERSION="$(metadata.get olm.replaces)"
CURRENT_CSV="$(metadata.get project.name).v$CURRENT_VERSION"
PREVIOUS_CSV="$(metadata.get project.name).v$PREVIOUS_VERSION"
export CURRENT_VERSION PREVIOUS_VERSION CURRENT_CSV PREVIOUS_CSV

# Directories below are filled with source code by ci-operator
export KNATIVE_SERVING_HOME="${GOPATH}/src/knative.dev/serving"
export KNATIVE_EVENTING_HOME="${GOPATH}/src/knative.dev/eventing"
export KNATIVE_EVENTING_KAFKA_BROKER_HOME="${GOPATH}/src/knative.dev/eventing-kafka-broker"
export KNATIVE_EVENTING_ISTIO_HOME="${GOPATH}/src/knative.dev/eventing-istio"
export BROKER_CLASS=${BROKER_CLASS:-"Kafka"}

export DOCKER_REPO_OVERRIDE="${DOCKER_REPO_OVERRIDE:-}"
export INTERACTIVE="${INTERACTIVE:-$(test -z "${GDMSESSION}"; echo $?)}"
export KUBECONFIG="${KUBECONFIG:-$(realpath ~/.kube/config)}"
export OPENSHIFT_CI="${OPENSHIFT_CI:-}"
export OPERATOR="${OPERATOR:-serverless-operator}"
export SCALE_UP="${SCALE_UP:--1}"

export OLM_NAMESPACE="${OLM_NAMESPACE:-openshift-marketplace}"
export OPERATORS_NAMESPACE="${OPERATORS_NAMESPACE:-openshift-serverless}"
export SERVING_NAMESPACE="${SERVING_NAMESPACE:-knative-serving}"
export INGRESS_NAMESPACE="${INGRESS_NAMESPACE:-knative-serving-ingress}"
export EVENTING_NAMESPACE="${EVENTING_NAMESPACE:-knative-eventing}"
# eventing e2e and conformance tests use a container for tracing tests that has hardcoded `istio-system` in it
export TRACING_NAMESPACE="${TRACING_NAMESPACE:-istio-system}"
export TRACING_BACKEND="${TRACING_BACKEND:-otel}"

declare -a SYSTEM_NAMESPACES
SYSTEM_NAMESPACES=("${TRACING_NAMESPACE}" "${OPERATORS_NAMESPACE}")
export SYSTEM_NAMESPACES
export UPGRADE_SERVERLESS="${UPGRADE_SERVERLESS:-"true"}"
export UPGRADE_CLUSTER="${UPGRADE_CLUSTER:-"false"}"
# Change this when forcing the upgrade to an image that is not yet available via upgrade channel
export UPGRADE_OCP_IMAGE="${UPGRADE_OCP_IMAGE:-}"

export INSTALL_PREVIOUS_VERSION="${INSTALL_PREVIOUS_VERSION:-"false"}"
export INSTALL_OLDEST_COMPATIBLE="${INSTALL_OLDEST_COMPATIBLE:-"false"}"

# Using first channel on the list, instead of default one
OLM_CHANNEL="${OLM_CHANNEL:-$(metadata.get 'olm.channels.list[*]' | head -n 1)}"
export OLM_CHANNEL
# Change this when upgrades need switching to a different channel
export OLM_UPGRADE_CHANNEL="${OLM_UPGRADE_CHANNEL:-"$OLM_CHANNEL"}"
export OLM_SOURCE="${OLM_SOURCE:-"$OPERATOR"}"
export TEST_KNATIVE_UPGRADE="${TEST_KNATIVE_UPGRADE:-true}"
export TEST_KNATIVE_E2E="${TEST_KNATIVE_E2E:-true}"
export TEST_KNATIVE_SERVING="${TEST_KNATIVE_SERVING:-false}"
export TEST_KNATIVE_EVENTING="${TEST_KNATIVE_EVENTING:-false}"
export TEST_KNATIVE_KAFKA="${TEST_KNATIVE_KAFKA:-false}"
export TEST_KNATIVE_KAFKA_BROKER="${TEST_KNATIVE_KAFKA_BROKER:-false}"

# Makefile triggers for modular install
export INSTALL_SERVING="${INSTALL_SERVING:-true}"
export INSTALL_EVENTING="${INSTALL_EVENTING:-true}"
export INSTALL_KAFKA="${INSTALL_KAFKA:-false}"
export FULL_MESH="${FULL_MESH:-false}"
export ENABLE_TRACING="${ENABLE_TRACING:-false}"
# Define sample-rate for tracing.
export SAMPLE_RATE="${SAMPLE_RATE:-"1.0"}"
export ZIPKIN_DEDICATED_NODE="${ZIPKIN_DEDICATED_NODE:-false}"
export QUAY_REGISTRY=quay.io/openshift-knative
DEFAULT_IMAGE_TEMPLATE=$(
  cat <<-EOF
{{- with .Name }}
{{- if eq . "httpproxy" }}${QUAY_REGISTRY}/serving/{{.}}:${KNATIVE_SERVING_VERSION#knative-}
{{- else if eq . "recordevents" }}${QUAY_REGISTRY}/eventing/{{.}}:${KNATIVE_EVENTING_VERSION#knative-}
{{- else if eq . "wathola-forwarder" }}${QUAY_REGISTRY}/eventing/{{.}}:${KNATIVE_EVENTING_VERSION#knative-}
{{- else if eq . "kafka" }}quay.io/strimzi/kafka:latest-kafka-3.4.0
{{- else }}${QUAY_REGISTRY}/{{.}}:multiarch{{end -}}
{{end -}}
EOF
)
export IMAGE_TEMPLATE="${IMAGE_TEMPLATE:-"$DEFAULT_IMAGE_TEMPLATE"}"
export INDEX_IMAGE="${INDEX_IMAGE:-}"
# Might be used to disable tests which need additional users.
# Managed environments such as Hypershift might now allow creating new users.
export USER_MANAGEMENT_ALLOWED="${USER_MANAGEMENT_ALLOWED:-true}"
export DELETE_CRD_ON_TEARDOWN="${DELETE_CRD_ON_TEARDOWN:-true}"
