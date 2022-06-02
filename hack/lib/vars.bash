#!/usr/bin/env bash

export BUILD_NUMBER=${BUILD_NUMBER:-$(head -c 128 < /dev/urandom | base64 | fold -w 8 | head -n 1)}

if [[ -n "${ARTIFACT_DIR:-}" ]]; then
  ARTIFACTS="${ARTIFACT_DIR}/build-${BUILD_NUMBER}"
  export ARTIFACTS
  mkdir -p "${ARTIFACTS}"
fi

# shellcheck disable=SC1091,SC1090
source "$(dirname "${BASH_SOURCE[0]}")/../../vendor/knative.dev/hack/e2e-tests.sh"

export STRIMZI_VERSION=0.28.0

# Adjust these when upgrading the knative versions.
export KNATIVE_SERVING_VERSION="${KNATIVE_SERVING_VERSION:-v$(metadata.get dependencies.serving)}"
export KNATIVE_EVENTING_VERSION="${KNATIVE_EVENTING_VERSION:-v$(metadata.get dependencies.eventing)}"
# TODO(matzew): SRVKE-1076 remove this when kafka e2e tests have been migrated
export KNATIVE_EVENTING_KAFKA_VERSION="${KNATIVE_EVENTING_KAFKA_VERSION:-v$(metadata.get dependencies.eventing_kafka)}"
export KNATIVE_EVENTING_KAFKA_BROKER_VERSION="${KNATIVE_EVENTING_KAFKA_BROKER_VERSION:-v$(metadata.get dependencies.eventing_kafka_broker)}"

CURRENT_VERSION="$(metadata.get project.version)"
PREVIOUS_VERSION="$(metadata.get olm.replaces)"
CURRENT_CSV="$(metadata.get project.name).v$CURRENT_VERSION"
PREVIOUS_CSV="$(metadata.get project.name).v$PREVIOUS_VERSION"
export CURRENT_VERSION PREVIOUS_VERSION CURRENT_CSV PREVIOUS_CSV

# Directories below are filled with source code by ci-operator
export KNATIVE_SERVING_HOME="${GOPATH}/src/knative.dev/serving"
export KNATIVE_EVENTING_HOME="${GOPATH}/src/knative.dev/eventing"
export KNATIVE_EVENTING_KAFKA_HOME="${GOPATH}/src/knative.dev/eventing-kafka"

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


# Using first channel on the list, instead of default one
OLM_CHANNEL="${OLM_CHANNEL:-$(metadata.get 'olm.channels.list[*]' | head -n 1)}"
export OLM_CHANNEL
# Change this when upgrades need switching to a different channel
export OLM_UPGRADE_CHANNEL="${OLM_UPGRADE_CHANNEL:-"$OLM_CHANNEL"}"
export OLM_SOURCE="${OLM_SOURCE:-"$OPERATOR"}"
export TEST_KNATIVE_UPGRADE="${TEST_KNATIVE_UPGRADE:-true}"
export TEST_KNATIVE_E2E="${TEST_KNATIVE_E2E:-true}"
export TEST_KNATIVE_KAFKA="${TEST_KNATIVE_KAFKA:-false}"

# Makefile triggers for modular install
export INSTALL_SERVING="${INSTALL_SERVING:-true}"
export INSTALL_EVENTING="${INSTALL_EVENTING:-true}"
export INSTALL_KAFKA="${INSTALL_KAFKA:-false}"
export FULL_MESH="${FULL_MESH:-false}"
export ENABLE_TRACING="${ENABLE_TRACING:-false}"
