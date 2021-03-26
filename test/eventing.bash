#!/usr/bin/env bash

readonly EVENTING_READY_FILE="/tmp/eventing-prober-ready"
readonly EVENTING_PROBER_FILE="/tmp/eventing-prober-signal"

function upstream_knative_eventing_e2e {
  logger.info 'Running eventing tests'

  local failed=0

  export TEST_IMAGE_TEMPLATE="registry.ci.openshift.org/openshift/knative-${KNATIVE_EVENTING_VERSION}:knative-eventing-test-{{.Name}}"

  cd "${KNATIVE_EVENTING_HOME}" || return $?

  # shellcheck disable=SC1090
  source "${KNATIVE_EVENTING_HOME}/openshift/e2e-common.sh"

  # run_e2e_tests defined in knative-eventing
  run_e2e_tests || failed=$?

  return $failed
}

function actual_eventing_version {
  oc get knativeeventing.operator.knative.dev \
    knative-eventing -n "${EVENTING_NAMESPACE}" -o=jsonpath="{.status.version}" \
    || return $?
}

function prepare_knative_eventing_tests {
  logger.info 'Nothing to prepare for Eventing upgrade tests'
}

function run_eventing_preupgrade_test {
  logger.info 'Running Eventing pre upgrade tests'
  local channels
  channels="${1:?Pass the list of channels as arg[1]}"

  cd "${KNATIVE_EVENTING_HOME}" || return $?

  local image_template
  # FIXME: SRVKE-606 use registry.ci.openshift.org image
  image_template="quay.io/openshift-knative/{{.Name}}:${KNATIVE_EVENTING_VERSION}"

  go_test_e2e -tags=preupgrade \
    -timeout=10m ./test/upgrade \
    -channels="${channels}" \
    --imagetemplate="${image_template}" \
    || return $?

  logger.success 'Eventing pre upgrade tests passed'
}

function start_eventing_prober {
  local eventing_prober_pid pid_file image_template eventing_prober_interval
  pid_file="${1:?Pass a PID file as arg[1]}"
  logger.info 'Starting Eventing prober'

  EVENTING_PROBER_INTERVAL_MSEC="${EVENTING_PROBER_INTERVAL_MSEC:-50}"
  eventing_prober_interval="${EVENTING_PROBER_INTERVAL_MSEC}ms"


  rm -fv "${EVENTING_PROBER_FILE}" "${EVENTING_READY_FILE}"
  cd "${KNATIVE_EVENTING_HOME}" || return $?

  # FIXME: SRVKE-606 use registry.ci.openshift.org image
  image_template="quay.io/openshift-knative/{{.Name}}:${KNATIVE_EVENTING_VERSION}"

  # FIXME: knative/operator#297 Restore scale to zero setting
  E2E_UPGRADE_TESTS_SERVING_SCALETOZERO=false \
  E2E_UPGRADE_TESTS_SERVING_USE=true \
  E2E_UPGRADE_TESTS_CONFIGMOUNTPOINT=/.config/wathola \
  E2E_UPGRADE_TESTS_INTERVAL="${eventing_prober_interval}" \
  go_test_e2e -tags=probe \
    -timeout=30m \
    ./test/upgrade \
    --pipefile="${EVENTING_PROBER_FILE}" \
    --readyfile="${EVENTING_READY_FILE}" \
    --imagetemplate="${image_template}" &
  eventing_prober_pid=$!

  logger.debug "Eventing prober PID is ${eventing_prober_pid}"

  echo ${eventing_prober_pid} > "${pid_file}"
}

function wait_for_eventing_prober_ready {
  wait_for_file "${EVENTING_READY_FILE}" || return $?

  logger.success 'Eventing prober is ready'
}

function end_eventing_prober {
  local prober_pid
  prober_pid="${1:?Pass a prober pid as arg[1]}"

  end_prober 'Eventing' "${prober_pid}" "${EVENTING_PROBER_FILE}" || return $?
}

function check_eventing_upgraded {
  local latest_version
  latest_version="${1:?Pass a target eventing version as arg[1]}"

  logger.debug 'Check KnativeEventing has the latest version with Ready status'
  timeout 300 "[[ ! ( \$(oc get knativeeventing.operator.knative.dev \
    knative-eventing -n ${EVENTING_NAMESPACE} -o=jsonpath='{.status.version}') \
    == ${latest_version} && \$(oc get knativeeventing.operator.knative.dev \
    knative-eventing -n ${EVENTING_NAMESPACE} \
    -o=jsonpath='{.status.conditions[?(@.type==\"Ready\")].status}') == True ) ]]" \
    || return $?
}

function run_eventing_postupgrade_test {
  logger.info 'Running Eventing post upgrade tests'
  local image_template channels
  channels="${1:?Pass the list of channels as arg[1]}"

  cd "${KNATIVE_EVENTING_HOME}" || return $?

  # FIXME: SRVKE-606 use registry.ci.openshift.org image
  image_template="quay.io/openshift-knative/{{.Name}}:${KNATIVE_EVENTING_VERSION}"

  go_test_e2e -tags=postupgrade \
    -timeout=10m ./test/upgrade \
    -channels="${channels}" \
    --imagetemplate="${image_template}" \
    || return $?

  logger.success 'Eventing post upgrade tests passed'
}
