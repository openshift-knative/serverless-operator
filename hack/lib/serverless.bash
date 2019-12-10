#!/usr/bin/env bash

function ensure_serverless_installed {
  logger.info 'Check if Serverless is installed'
  if oc get knativeserving knative-serving -n "${SERVING_NAMESPACE}" >/dev/null 2>&1; then
    logger.success 'Serverless is already installed.'
    return 0
  fi
  install_serverless_latest
}

function install_serverless_previous {
  local rootdir current_csv previous_csv
  rootdir="$(dirname "$(dirname "$(dirname "$(realpath "${BASH_SOURCE[0]}")")")")"

  # Remove installplan from previous installations, leaving this would make the operator
  # upgrade to the latest version immediately
  current_csv=$("${rootdir}/hack/catalog.sh" | grep currentCSV | awk '{ print $2 }')
  remove_installplan "$current_csv"

  previous_csv=$("${rootdir}/hack/catalog.sh" | grep replaces: | tail -n1 | awk '{ print $2 }')
  deploy_serverless_operator "$previous_csv"  || return $?
  deploy_knativeserving_cr || return $?
}

function remove_installplan {
  local install_plan
  install_plan=$(find_install_plan $1)
  if [[ -n $install_plan ]]; then
    oc delete "$install_plan" -n "${OPERATORS_NAMESPACE}"
  fi
}

function install_serverless_latest {
  deploy_serverless_operator_latest || return $?
  deploy_knativeserving_cr || return $?
}

function deploy_serverless_operator_latest {
  local rootdir csv
  rootdir="$(dirname "$(dirname "$(dirname "$(realpath "${BASH_SOURCE[0]}")")")")"
  # Get the current/latest CSV
  csv=$("${rootdir}/hack/catalog.sh" | grep currentCSV | awk '{ print $2 }')

  deploy_serverless_operator "${csv}"
}

function deploy_serverless_operator {
  logger.info 'Install the Serverless Operator'
  local csv
  csv="$1"

  cat <<EOF | oc apply -f - || return $?
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: ${OPERATOR}
  namespace: ${OPERATORS_NAMESPACE}
spec:
  channel: techpreview
  name: ${OPERATOR}
  source: ${OPERATOR}
  sourceNamespace: ${OLM_NAMESPACE}
  installPlanApproval: Manual
  startingCSV: ${csv}
EOF

  # Approve the initial installplan automatically
  approve_csv "$csv" || return 5
}

function approve_csv {
  local csv_version install_plan
  csv_version=$1

  # Wait for the installplan to be available
  timeout 900 "[[ -z \$(find_install_plan $csv_version) ]]" || return 1

  install_plan=$(find_install_plan $csv_version)
  oc get $install_plan -n ${OPERATORS_NAMESPACE} -o yaml | sed 's/\(.*approved:\) false/\1 true/' | oc replace -f -

  timeout 300 "[[ \$(oc get ClusterServiceVersion $csv_version -n ${OPERATORS_NAMESPACE} -o jsonpath='{.status.phase}') != Succeeded ]]" || return 1
}

function find_install_plan {
  local csv=$1
  for plan in `oc get installplan -n ${OPERATORS_NAMESPACE} --no-headers -o name`; do 
    [[ $(oc get $plan -n ${OPERATORS_NAMESPACE} -o=jsonpath='{.spec.clusterServiceVersionNames}' | grep -c $csv) -eq 1 && \
       $(oc get $plan -n ${OPERATORS_NAMESPACE} -o=jsonpath="{.status.catalogSources}" | grep -c $OPERATOR) -eq 1 ]] && echo $plan && return 0
  done
  echo ""
}

function deploy_knativeserving_cr {
  logger.info 'Deploy Knative Serving'

  # Wait for the CRD to appear
  timeout 900 "[[ \$(oc get crd | grep -c knativeservings) -eq 0 ]]" || return 6

  # Install Knative Serving
  cat <<EOF | oc apply -f - || return $?
apiVersion: serving.knative.dev/v1alpha1
kind: KnativeServing
metadata:
  name: knative-serving
  namespace: ${SERVING_NAMESPACE}
EOF

  timeout 900 '[[ $(oc get knativeserving knative-serving -n $SERVING_NAMESPACE -o=jsonpath="{.status.conditions[?(@.type==\"Ready\")].status}") != True ]]'  || return 7

  logger.success 'Serverless has been installed sucessfully.'
}

function teardown_serverless {
  logger.warn '😭  Teardown Serverless...'

  if oc get knativeserving knative-serving -n "${SERVING_NAMESPACE}" >/dev/null 2>&1; then
    logger.info 'Removing KnativeServing CR'
    oc delete knativeserving knative-serving -n "${SERVING_NAMESPACE}" || return $?

    logger.info 'Wait until there are no knative serving pods running'
    timeout 600 "[[ \$(oc get pods -n ${SERVING_NAMESPACE} -o jsonpath='{.items}') != '[]' ]]" || return 9
  fi

  logger.success 'Serverless has been uninstalled.'
}
