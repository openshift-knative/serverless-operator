#!/usr/bin/env bash

# Copyright 2018 The Knative Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# This script provides helper methods to perform cluster actions.
source $(dirname $0)/../vendor/github.com/knative/test-infra/scripts/e2e-tests.sh

# Latest serving release. This is intentionally hardcoded for now, but
# will need the ability to test against the latest successful serving
# CI runs in the future.
readonly LATEST_SERVING_RELEASE_VERSION=0.8.1
# Istio version we test with
readonly ISTIO_VERSION=1.1.3
# Test without Istio mesh enabled
readonly ISTIO_MESH=0
# Namespace used for tests
readonly TEST_NAMESPACE="operator-tests"

# Choose a correct istio-crds.yaml file.
# - $1 specifies Istio version.
function istio_crds_yaml() {
  local istio_version="$1"
  echo "third_party/istio-${istio_version}/istio-crds.yaml"
}

# Choose a correct istio.yaml file.
# - $1 specifies Istio version.
# - $2 specifies whether we should use mesh.
function istio_yaml() {
  local istio_version="$1"
  local istio_mesh=$2
  local suffix=""
  if [[ $istio_mesh -eq 0 ]]; then
    suffix="-lean"
  fi
  echo "third_party/istio-${istio_version}/istio${suffix}.yaml"
}

# Install Istio.
function install_istio() {
  local base_url="https://raw.githubusercontent.com/knative/serving/v${LATEST_SERVING_RELEASE_VERSION}"
  # Decide the Istio configuration to install.
  if [[ -z "$ISTIO_VERSION" ]]; then
    # Defaults to 1.1-latest
    ISTIO_VERSION=1.1-latest
  fi
  if [[ -z "$ISTIO_MESH" ]]; then
    # Defaults to using mesh.
    ISTIO_MESH=1
  fi
  INSTALL_ISTIO_CRD_YAML="${base_url}/$(istio_crds_yaml $ISTIO_VERSION)"
  INSTALL_ISTIO_YAML="${base_url}/$(istio_yaml $ISTIO_VERSION $ISTIO_MESH)"

  echo ">> Installing Istio"
  echo "Istio CRD YAML: ${INSTALL_ISTIO_CRD_YAML}"
  echo "Istio YAML: ${INSTALL_ISTIO_YAML}"
    
  echo ">> Bringing up Istio"
  echo ">> Running Istio CRD installer"
  kubectl apply -f "${INSTALL_ISTIO_CRD_YAML}" || return 1
  wait_until_batch_job_complete istio-system || return 1

  echo ">> Running Istio"
  kubectl apply -f "${INSTALL_ISTIO_YAML}" || return 1
}

function install_serving_operator() {
  header "Installing Knative Serving operator"

  # Deploy the operator
  kubectl create ns knative-serving
  kubectl apply -f deploy/crds/serving_v1alpha1_knativeserving_crd.yaml
  kubectl apply -f deploy/

  # Install Knative Serving
  kubectl apply -n knative-serving -f deploy/crds/serving_v1alpha1_knativeserving_cr.yaml

  # Wait for Serving to come up
  sleep 10
  wait_until_pods_running knative-serving
}

function knative_setup() {
  install_istio || fail_test "Istio installation failed"
  install_serving_operator
}

# Create test resources
function test_setup() {
  echo ">> Creating test namespaces"
  kubectl create namespace $TEST_NAMESPACE
}

# Delete test resources
function test_teardown() {
  echo ">> Removing test namespaces"
  kubectl delete all --all --ignore-not-found --now --timeout 60s -n $TEST_NAMESPACE
  kubectl delete --ignore-not-found --now --timeout 300s namespace $TEST_NAMESPACE
}

# Uninstalls Knative Serving from the current cluster.
function knative_teardown() {
  echo ">> Uninstalling Knative serving"
  echo "Istio YAML: ${INSTALL_ISTIO_YAML}"
  echo ">> Bringing down Serving"
  kubectl delete -n knative-serving knativeserving --all
  echo ">> Bringing down Istio"
  kubectl delete --ignore-not-found=true -f "${INSTALL_ISTIO_YAML}" || return 1
  kubectl delete --ignore-not-found=true clusterrolebinding cluster-admin-binding
}

function dump_extra_cluster_state() {
  kubectl get nodes -oyaml
}
