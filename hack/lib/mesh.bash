#!/usr/bin/env bash

resources_dir="$(dirname "${BASH_SOURCE[0]}")/mesh_resources"
  
mesh_deployments=(istio-operator jaeger-operator kiali-operator)

function install_mesh {
  deploy_servicemesh_operators
  if [[ ${FULL_MESH:-} == "true" ]]; then
     deploy_servicemeshcontrolplane
     deploy_gateways
  fi
}

function uninstall_mesh {
  if [[ ${FULL_MESH:-} == "true" ]]; then
     undeploy_gateways
     undeploy_servicemeshcontrolplane
  fi
  undeploy_servicemesh_operators
}

function deploy_servicemesh_operators {
  logger.info "Installing service mesh operators in namespace openshift-operators"
  oc apply -f ${resources_dir}/subscription.yaml || return $?

  logger.info "Waiting until service mesh operators are available"
  timeout 600 "[[ \$(oc get deploy -n openshift-operators ${mesh_deployments[*]} --no-headers | wc -l) != 3 ]]" || return 1
  oc wait --for=condition=Available deployment "${mesh_deployments[@]}" --timeout=300s -n openshift-operators || return $?
}

function undeploy_servicemesh_operators {
  logger.info "Deleting service mesh subscriptions"
  oc delete subscriptions.operators.coreos.com -n openshift-operators servicemeshoperator kiali-ossm jaeger-product --ignore-not-found
}

function deploy_servicemeshcontrolplane {
  logger.info "Installing ServiceMeshControlPlane in namespace istio-system"

  oc get ns istio-system || oc create namespace istio-system

  # Make sure servicemeshcontrolplanes.maistra.io is available.
  timeout 120 "[[ \$(oc get crd servicemeshcontrolplanes.maistra.io --no-headers | wc -l) != 1 ]]" || return 1
  oc wait --for=condition=Established crd servicemeshcontrolplanes.maistra.io

  # creating smcp often fails due to webhook error
  timeout 120 "[[ \$(oc apply -f ${resources_dir}/smcp.yaml | oc get smcp -n istio-system basic --no-headers | wc -l) != 1 ]]" || return 1
  oc wait --timeout=180s --for=condition=Ready smcp -n istio-system basic || oc get smcp -n istio-system basic -o yaml
}

function undeploy_servicemeshcontrolplane {
  logger.info "Deleting ServiceMeshControlPlane"
  oc delete smcp -n openshift-operators basic --ignore-not-found || return $?
}

function deploy_gateways {
  oc apply -f ${resources_dir}/smmr.yaml || return $?

  local out_dir=$(mktemp -d /tmp/certs-XXX)

  # Generate wildcard certs with cluster's subdomain.

  oc extract -n openshift-apiserver configmap/config --to=${out_dir}
  subdomain=$(yq read ${out_dir}/config.yaml "routingConfig.subdomain")

  openssl req -x509 -sha256 -nodes -days 365 -newkey rsa:2048 \
    -subj "/O=Example Inc./CN=$subdomain" \
    -keyout ${out_dir}/root.key \
    -out ${out_dir}/root.crt

  openssl req -nodes -newkey rsa:2048 \
      -subj "/CN=*.${subdomain}/O=Example Inc." \
      -keyout ${out_dir}/wildcard.key \
      -out ${out_dir}/wildcard.csr

  openssl x509 -req -days 365 -set_serial 0 \
      -CA ${out_dir}/root.crt \
      -CAkey ${out_dir}/root.key \
      -in ${out_dir}/wildcard.csr \
      -out ${out_dir}/wildcard.crt

  oc create -n istio-system secret tls wildcard-certs \
      --key=${out_dir}/wildcard.key \
      --cert=${out_dir}/wildcard.crt --dry-run=client -o yaml | oc apply -f - 

  # ca-key-pair secret in cert-manager namespace needs for upstream e2e test with https option.
  oc get ns cert-manager || oc create namespace cert-manager
  oc create -n cert-manager secret tls ca-key-pair \
      --key=${out_dir}/wildcard.key \
      --cert=${out_dir}/wildcard.crt --dry-run=client -o yaml | oc apply -f -

  oc apply -f ${resources_dir}/gateway.yaml || return $?
  oc apply -f ${resources_dir}/peerauthentication.yaml || return $?
}

function undeploy_gateways {
  oc delete -f ${resources_dir}/peerauthentication.yaml --ignore-not-found || return $?
  oc delete -f ${resources_dir}/gateway.yaml --ignore-not-found || return $?
  oc delete -f ${resources_dir}/smmr.yaml --ignore-not-found || return $?
  oc delete -n cert-manager secret ca-key-pair  --ignore-not-found || return $?
  oc delete -n istio-system secret wildcard-certs --ignore-not-found || return $?
}

function deploy_net_istio {
  oc apply -f ${resources_dir}/knativeserving.yaml || return $?
}

function undeploy_net_istio {
  oc delete -f ${resources_dir}/knativeserving.yaml --ignore-not-found || return $?
}
