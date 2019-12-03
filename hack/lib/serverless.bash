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
  local rootdir="$(dirname "$(dirname "$(dirname "$(realpath "${BASH_SOURCE[0]}")")")")"
  # Get the previous CSV
  local csv=$(${rootdir}/hack/catalog.sh | grep replaces: | tail -n1 | awk '{ print $2 }')

  deploy_serverless_operator $csv  || return $?
  deploy_knativeserving_cr || return $?
}

function install_serverless_latest {
  local rootdir="$(dirname "$(dirname "$(dirname "$(realpath "${BASH_SOURCE[0]}")")")")"
  # Get the current/latest CSV
  local csv=$(${rootdir}/hack/catalog.sh | grep currentCSV | awk '{ print $2 }')

  deploy_serverless_operator $csv || return $?
  deploy_knativeserving_cr || return $?
}

function deploy_serverless_operator {
  logger.info 'Install the Serverless Operator'
  local csv=$1

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
  sourceNamespace: ${OPERATORS_NAMESPACE}
  installPlanApproval: Manual
  startingCSV: ${csv}
EOF

  # Approve the initial installplan automatically
  approve_csv $csv

  logger.info "Wait for the ${OPERATOR} pod to appear"
  timeout 900 "[[ \$(oc get pods -n ${OPERATORS_NAMESPACE} | grep -c ${OPERATOR}) -eq 0 ]]" || return 5

  logger.info 'Wait until the Operator pod is up and running'
  wait_until_pods_running "${OPERATORS_NAMESPACE}" || return 6

  logger.info 'Wait until Operator finishes installation'
  timeout 300 "[[ \$(oc get csv -n ${OPERATORS_NAMESPACE} | grep ${OPERATOR} | grep -c Succeeded) -eq 0 ]]" || return 7
}

function approve_csv {
  local csv_version=$1

  # Wait for the installplan to be available
  timeout 900 "[[ -z \$(find_install_plan $csv_version) ]]" || return 1

  local install_plan=$(find_install_plan $csv_version)
  oc get $install_plan -n ${OPERATORS_NAMESPACE} -o yaml | sed 's/\(.*approved:\) false/\1 true/' | oc replace -f -

  timeout 300 "[[ \$(oc get ClusterServiceVersion $csv_version -n ${OPERATORS_NAMESPACE} -o jsonpath='{.status.phase}') != Succeeded ]]" || return 1
}

function find_install_plan {
  local csv=$1
  for plan in `oc get installplan -n ${OPERATORS_NAMESPACE} --no-headers -o name`; do 
    [[ $(oc get $plan -n ${OPERATORS_NAMESPACE} -o=jsonpath='{.spec.clusterServiceVersionNames}' | grep -c $csv) -eq 1 && \
       $(oc get $plan -n ${OPERATORS_NAMESPACE} -o=jsonpath="{.metadata.ownerReferences[?(@.name==\"${OPERATOR}\")]}") != "" ]] && echo $plan && return 0
  done
  echo ""
}

function deploy_knativeserving_cr {
  logger.info 'Deploy Knative Serving'

  # Install Knative Serving
  cat <<EOF | oc apply -f - || return $?
apiVersion: serving.knative.dev/v1alpha1
kind: KnativeServing
metadata:
  name: knative-serving
  namespace: ${SERVING_NAMESPACE}
EOF

  timeout 900 '[[ $(oc get knativeserving knative-serving -n $SERVING_NAMESPACE -o=jsonpath="{.status.conditions[?(@.type==\"Ready\")].status}") != True ]]'  || return 8

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
