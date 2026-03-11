#!/usr/bin/env bash

mesh_v3_resources_dir="$(dirname "${BASH_SOURCE[0]}")/mesh_v3_resources"

function install_mesh3 {
  ensure_catalog_pods_running
  deploy_sail_operator
  deploy_istio
  deploy_mesh3_gateways
}

function uninstall_mesh3 {
  undeploy_mesh3_gateways
  undeploy_istio
  undeploy_sail_operator
}

function deploy_sail_operator {
  if [[ ${SKIP_OPERATOR_SUBSCRIPTION:-} != "true" ]]; then
    logger.info "Installing Service Mesh 3 operator in namespace openshift-operators"
    oc apply -f "${mesh_v3_resources_dir}"/01_subscription.yaml || return $?
  fi

  logger.info "Waiting until Service Mesh 3 operator is available"
  timeout 600 "[[ \$(oc get deploy -n openshift-operators servicemesh-operator3 --no-headers 2>/dev/null | wc -l) != 1 ]]" || return 1
  oc wait --for=condition=Available deployment servicemesh-operator3 --timeout=300s -n openshift-operators || return $?
}

function undeploy_sail_operator {
  logger.info "Deleting Service Mesh 3 operator subscription"
  oc delete subscriptions.operators.coreos.com -n openshift-operators servicemeshoperator3 --ignore-not-found

  logger.info 'Deleting ClusterServiceVersion'
  for csv in $(set +o pipefail && oc get csv -n openshift-operators --no-headers 2>/dev/null \
      | grep 'servicemeshoperator3' | cut -f1 -d' '); do
    oc delete csv -n openshift-operators "${csv}"
  done

  logger.info 'Ensure no operators present'
  timeout 600 "[[ \$(oc get deployments -n openshift-operators -oname | grep -c 'servicemeshoperator3') != 0 ]]"

  logger.info 'Ensure no CRDs left'
  if [[ ! $(oc get crd -oname | grep -c 'sailoperator.io') -eq 0 ]]; then
    oc get crd -oname | grep 'sailoperator.io' | xargs oc delete --timeout=60s
  fi
  logger.success "Service Mesh 3 operator has been uninstalled"
}

function deploy_istio {
  logger.info "Installing Istio and IstioCNI"

  # Make sure istios.sailoperator.io CRD is available.
  timeout 120 "[[ \$(oc get crd istios.sailoperator.io --no-headers 2>/dev/null | wc -l) != 1 ]]" || return 1
  oc wait --for=condition=Established crd istios.sailoperator.io

  # Create namespaces for Istio and IstioCNI.
  oc get ns istio-system || oc create namespace istio-system
  oc get ns istio-cni || oc create namespace istio-cni

  # Substitute the MESH3_ISTIO_VERSION placeholder and apply Istio CR.
  local istio_cr
  istio_cr="$(mktemp -t istio-XXXXX.yaml)"
  sed "s/MESH3_ISTIO_VERSION/${MESH3_ISTIO_VERSION}/g" "${mesh_v3_resources_dir}/02_istio.yaml" > "${istio_cr}"
  oc apply -f "${istio_cr}" -n istio-system || return $?

  # Substitute the MESH3_ISTIO_VERSION placeholder and apply IstioCNI CR.
  local istiocni_cr
  istiocni_cr="$(mktemp -t istiocni-XXXXX.yaml)"
  sed "s/MESH3_ISTIO_VERSION/${MESH3_ISTIO_VERSION}/g" "${mesh_v3_resources_dir}/03_istiocni.yaml" > "${istiocni_cr}"
  oc apply -f "${istiocni_cr}" -n istio-cni || return $?

  timeout 120 "[[ \$(oc get istio -n istio-system default --no-headers 2>/dev/null | wc -l) != 1 ]]" || return 1

  oc wait --timeout=180s --for=condition=Ready istio -n istio-system default || oc get istio -n istio-system default -o yaml
  oc wait --timeout=180s --for=condition=Ready istiocni -n istio-cni default || oc get istiocni -n istio-cni default -o yaml

  rm -f "${istio_cr}" "${istiocni_cr}"
}

function undeploy_istio {
  logger.info "Deleting Istio and IstioCNI"
  oc delete istiocni -n istio-cni default --ignore-not-found || return $?
  oc delete istio -n istio-system default --ignore-not-found || return $?
}

function deploy_mesh3_gateways {
  # Generate wildcard certs with cluster's subdomain.
  local out_dir
  out_dir="$(mktemp -d /tmp/certs-XXX)"

  openssl req -x509 -sha256 -nodes -days 365 -newkey rsa:2048 \
    -subj "/O=Example Inc./CN=Example" \
    -keyout "${out_dir}"/root.key \
    -out "${out_dir}"/root.crt

  subdomain=$(oc get ingresses.config.openshift.io cluster -o jsonpath="{.spec.domain}")
  openssl req -nodes -newkey rsa:2048 \
      -subj "/O=Example Inc./CN=Example" \
      -reqexts san \
      -config <(printf "[req]\ndistinguished_name=req\n[san]\nsubjectAltName=DNS:*.%s" "$subdomain") \
      -keyout "${out_dir}"/wildcard.key \
      -out "${out_dir}"/wildcard.csr

  openssl x509 -req -days 365 -set_serial 0 \
      -extfile <(printf "subjectAltName=DNS:*.%s" "$subdomain") \
      -CA "${out_dir}"/root.crt \
      -CAkey "${out_dir}"/root.key \
      -in "${out_dir}"/wildcard.csr \
      -out "${out_dir}"/wildcard.crt

  oc get ns knative-serving-ingress || oc create namespace knative-serving-ingress

  # Wildcard certs go into knative-serving-ingress for SM3.
  oc create -n knative-serving-ingress secret tls wildcard-certs \
      --key="${out_dir}"/wildcard.key \
      --cert="${out_dir}"/wildcard.crt --dry-run=client -o yaml | oc apply -f -

  # ca-key-pair secret in cert-manager namespace needed for upstream e2e test with https option.
  oc get ns cert-manager || oc create namespace cert-manager
  oc create -n cert-manager secret tls ca-key-pair \
      --key="${out_dir}"/wildcard.key \
      --cert="${out_dir}"/wildcard.crt --dry-run=client -o yaml | oc apply -f -

  oc apply -f "${mesh_v3_resources_dir}"/04_namespace.yaml || return $?
  oc apply -f "${mesh_v3_resources_dir}"/05_gateway_deploy.yaml || return $?
  oc apply -f "${mesh_v3_resources_dir}"/06_serving_gateways.yaml || return $?
  oc apply -f "${mesh_v3_resources_dir}"/07_peer_authentication.yaml || return $?

  oc apply -f "${mesh_v3_resources_dir}"/authorization-policies/setup || return $?
  oc apply -f "${mesh_v3_resources_dir}"/authorization-policies/helm || return $?
}

function undeploy_mesh3_gateways {
  oc delete -f "${mesh_v3_resources_dir}"/authorization-policies/helm --ignore-not-found || return $?
  oc delete -f "${mesh_v3_resources_dir}"/authorization-policies/setup --ignore-not-found || return $?
  oc delete -f "${mesh_v3_resources_dir}"/07_peer_authentication.yaml --ignore-not-found || return $?
  oc delete -f "${mesh_v3_resources_dir}"/06_serving_gateways.yaml --ignore-not-found || return $?
  oc delete -f "${mesh_v3_resources_dir}"/05_gateway_deploy.yaml --ignore-not-found || return $?
  oc delete -n cert-manager secret ca-key-pair --ignore-not-found || return $?
  oc delete -n knative-serving-ingress secret wildcard-certs --ignore-not-found || return $?
}
