#!/usr/bin/env bash

certmanager_resources_dir="$(dirname "${BASH_SOURCE[0]}")/certmanager_resources"
  
function install_certmanager {
  ensure_catalog_pods_running
  deploy_certmanager_operator
  deploy_certificates
}

function uninstall_certmanager {
  undeploy_certmanager_operator
}

function deploy_certmanager_operator {
  logger.info "Installing cert manager operator"

  deployment_namespace="cert-manager"

  ocp_version=$(oc get clusterversion version -o jsonpath='{.status.desired.version}')
  # Workaround for cert-manager not being available in 4.18 yet.
  # https://issues.redhat.com/browse/SRVCOM-3428
  if versions.ge "$(versions.major_minor "$ocp_version")" "4.18"; then
    oc apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.15.3/cert-manager.yaml
  else
    oc apply -f "${certmanager_resources_dir}"/subscription.yaml || return $?
  fi

  logger.info "Waiting until cert manager operator is available"

  timeout 600 "[[ \$(oc get deploy -n ${deployment_namespace} cert-manager --no-headers | wc -l) != 1 ]]" || return 1
  timeout 600 "[[ \$(oc get deploy -n ${deployment_namespace} cert-manager-webhook --no-headers | wc -l) != 1 ]]" || return 1
  oc wait deployments -n ${deployment_namespace} cert-manager-webhook --for condition=available --timeout=600s
  oc wait deployments -n ${deployment_namespace} cert-manager --for condition=available --timeout=600s
}

function deploy_certificates {
  logger.info "Installing certificates"

  deployment_namespace="cert-manager"

  # serving resources
  oc apply -f "${certmanager_resources_dir}"/serving-selfsigned-issuer.yaml || return $?
  oc apply -f "${certmanager_resources_dir}"/serving-ca-issuer.yaml || return $?
  oc apply -n "${deployment_namespace}" -f "${certmanager_resources_dir}"/serving-ca-certificate.yaml || return $?

  sync_trust_bundle "knative-selfsigned-ca" "knative-serving" "knative-serving-ingress" || return $?
  if [[ $MESH == "true" ]]; then
    sync_trust_bundle "knative-selfsigned-ca" "istio-system" || return $?
  fi

  # eventing resources
  oc apply -f "${certmanager_resources_dir}"/selfsigned-issuer.yaml || return $?
  oc apply -f "${certmanager_resources_dir}"/eventing-ca-issuer.yaml || return $?
  oc apply -n "${deployment_namespace}" -f "${certmanager_resources_dir}"/ca-certificate.yaml || return $?
  sync_trust_bundle "knative-eventing-ca" "knative-eventing" || return $?
}

function undeploy_certmanager_operator {
  logger.info "Deleting cert manager subscriptions"
  oc delete -f "${certmanager_resources_dir}"/subscription.yaml || return $?

  logger.info 'Ensure no CRDs left'
  if [[ ! $(oc get crd -oname | grep -c 'maistra.io') -eq 0 ]]; then
    oc get crd -oname | grep 'cert-manager.io' | xargs oc delete --timeout=60s
  fi
  logger.success "cert manager has been uninstalled"
}

function sync_trust_bundle {
   logger.info "Syncing cert-manager CA to trust-bundle for Knative components"
   local ca_secret
   ca_secret="${1:?Pass CA secret name as arg[1]}"
   shift
   local namespaces=("${@}")

   wait_until_object_exists secret "${ca_secret}" "${deployment_namespace}" || return $?

   oc get secret -n "${deployment_namespace}" "${ca_secret}" -o=jsonpath='{.data.tls\.crt}' | base64 -d > tls.crt || return $?
   oc get secret -n "${deployment_namespace}" "${ca_secret}" -o=jsonpath='{.data.ca\.crt}' | base64 -d > ca.crt || return $?

   for ns in "${namespaces[@]}"; do
     echo "Syncing trust-bundle for namespace: ${ns}"
     oc create namespace "${ns}" --dry-run=client -o yaml | oc apply -f -
     oc label namespace "${ns}" knative.openshift.io/part-of="openshift-serverless" --overwrite
     oc create configmap -n "${ns}" knative-ca-bundle --from-file=tls.crt --from-file=ca.crt \
        --dry-run=client -o yaml | kubectl apply -n "${ns}" -f - || return $?
     oc label configmap -n "${ns}" knative-ca-bundle networking.knative.dev/trust-bundle=true --overwrite
   done

   rm -f tls.crt ca.crt
}
