#!/usr/bin/env bash

certmanager_resources_dir="$(dirname "${BASH_SOURCE[0]}")/certmanager_resources"
  
function install_certmanager {
  deploy_certmanager_operator
}

function uninstall_certmanager {
  undeploy_certmanager_operator
}

function deploy_certmanager_operator {
  logger.info "Installing cert manager operator"

  openshift_version=$(oc version -o yaml | yq read - openshiftVersion)
  deployment_namespace="cert-manager"
  if printf '%s\n4.12\n' "${openshift_version}" | sort --version-sort -C; then
      # OCP version is older as 4.12 and thus cert-manager-operator is only available as tech-preview in this version (cert-manager-operator GA'ed in OCP 4.12)
      
      echo "Running on OpenShift ${openshift_version} which supports cert-manager-operator only in tech-preview"

      yq delete "${certmanager_resources_dir}"/subscription.yaml --doc 1 spec | \
      yq write - --doc 2 spec.channel tech-preview | \
      oc apply -f - || return $?

      deployment_namespace="openshift-cert-manager"
  else
    echo "Running on OpenShift ${openshift_version} which supports GA'ed cert-manager-operator"

    oc apply -f "${certmanager_resources_dir}"/subscription.yaml || return $?
  fi

  logger.info "Waiting until cert manager operator is available"
  timeout 600 "[[ \$(oc get deploy -n ${deployment_namespace} cert-manager --no-headers | wc -l) != 1 ]]" || return 1
  timeout 600 "[[ \$(oc get deploy -n ${deployment_namespace} cert-manager-webhook --no-headers | wc -l) != 1 ]]" || return 1

  oc apply -f "${certmanager_resources_dir}"/selfsigned-issuer.yaml || return $?
  oc apply -f "${certmanager_resources_dir}"/eventing-ca-issuer.yaml || return $?
  oc apply -f "${certmanager_resources_dir}"/ca-certificate.yaml || return $?

  local ca_cert_tls_secret="knative-eventing-ca"
  echo "Waiting until secrets: ${ca_cert_tls_secret} exist in ${deployment_namespace}"
  wait_until_object_exists secret "${ca_cert_tls_secret}" "${deployment_namespace}" || return $?

  oc get secret -n "${deployment_namespace}" "${ca_cert_tls_secret}" -o=jsonpath='{.data.tls\.crt}' | base64 -d > tls.crt || return $?
  oc get secret -n "${deployment_namespace}" "${ca_cert_tls_secret}" -o=jsonpath='{.data.ca\.crt}' | base64 -d > ca.crt || return $?

  oc create namespace "${EVENTING_NAMESPACE}" --dry-run=client -o yaml | oc apply -f -
  oc create configmap -n "${EVENTING_NAMESPACE}" knative-eventing-bundle --from-file=tls.crt --from-file=ca.crt \
    --dry-run=client -o yaml | kubectl apply -n knative-eventing -f - || return $?

  oc label configmap -n "${EVENTING_NAMESPACE}" knative-eventing-bundle networking.knative.dev/trust-bundle=true
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
