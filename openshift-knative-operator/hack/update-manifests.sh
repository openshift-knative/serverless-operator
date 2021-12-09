#!/usr/bin/env bash

set -Eeuo pipefail

root="$(dirname "${BASH_SOURCE[0]}")/../.."

# Source the main vars file to get the serving/eventing version to be used.
# shellcheck disable=SC1091,SC1090
source "$root/hack/lib/__sources__.bash"

# These files could in theory change from release to release, though their names should
# be fairly stable.
serving_files=(serving-crds serving-core serving-hpa serving-post-install-jobs)
eventing_files=(eventing-crds eventing-core in-memory-channel mt-channel-broker eventing-sugar-controller eventing-post-install)

# This excludes the gateways and peerauthentication settings as we want customers to do
# manipulate those.
istio_files=(200-clusterrole 400-config-istio 500-controller 500-webhook-deployment 500-webhook-secret 500-webhook-service 600-mutating-webhook 600-validating-webhook)

function download {
  component=$1
  version=$2
  shift
  shift

  files=("$@")

  component_dir="$root/openshift-knative-operator/cmd/operator/kodata/knative-${component}"
  target_dir="${component_dir}/${version:1}"
  rm -r "$component_dir"
  mkdir -p "$target_dir"

  for (( i=0; i<${#files[@]}; i++ ));
  do
    index=$(( i+1 ))
    file="${files[$i]}.yaml"
    target_file="$target_dir/$index-$file"
    url="https://github.com/knative/$component/releases/download/$version/$file"

    wget --no-check-certificate "$url" -O "$target_file"

    # Break all image references so we know our overrides work correctly.
    yaml.break_image_references "$target_file"
  done
}

function download_ingress {
  component=$1
  version=$2
  shift
  shift

  files=("$@")

  ingress_dir="$root/openshift-knative-operator/cmd/operator/kodata/ingress/$(versions.major_minor "${KNATIVE_SERVING_VERSION}")"
  rm -r "$ingress_dir"
  mkdir -p "$ingress_dir"

  for (( i=0; i<${#files[@]}; i++ ));
  do
    index=$(( i+1 ))
    file="${files[$i]}.yaml"
    ingress_target_file="$ingress_dir/$index-$file"
    url="https://raw.githubusercontent.com/knative-sandbox/${component}/${version}/config/${file}"

    wget --no-check-certificate "$url" -O "$ingress_target_file"

    # Break all image references so we know our overrides work correctly.
    yaml.break_image_references "$ingress_target_file"
  done
}

#
# DOWNLOAD SERVING
#
download serving "$KNATIVE_SERVING_VERSION" "${serving_files[@]}"

# Drop namespace from manifest.
git apply "$root/openshift-knative-operator/hack/001-serving-namespace-deletion.patch"

# Extra role for downstream, so that users can get the autoscaling CM to fetch defaults.
git apply "$root/openshift-knative-operator/hack/002-openshift-serving-role.patch"

# TODO: Remove this once upstream fixed https://github.com/knative/operator/issues/376.
# See also https://issues.redhat.com/browse/SRVKS-670.
git apply "$root/openshift-knative-operator/hack/003-serving-pdb.patch"

download_ingress net-istio "v$(metadata.get dependencies.net_istio)" "${istio_files[@]}"

url="https://github.com/knative-sandbox/net-kourier/releases/download/v$(metadata.get dependencies.kourier)/kourier.yaml"
kourier_file="$root/openshift-knative-operator/cmd/operator/kodata/ingress/$(versions.major_minor "${KNATIVE_SERVING_VERSION}")/kourier.yaml"
wget --no-check-certificate "$url" -O "$kourier_file"
# TODO: [SRVKS-610] These values should be replaced by operator instead of sed.
sed -i -e 's/net-kourier-controller.knative-serving/net-kourier-controller/g' "$kourier_file"
# Break all image references so we know our overrides work correctly.
yaml.break_image_references "$kourier_file"

# Add networkpolicy for webhook when net-istio is enabled.
git apply "$root/openshift-knative-operator/hack/007-networkpolicy-mesh.patch"

# Make Kourier rollout in a more defensive way so no requests get dropped.
# TODO: Can probably be removed in 1.21 and/or be sent upstream.
git apply "$root/openshift-knative-operator/hack/008-kourier-rollout.patch"

#
# DOWNLOAD EVENTING
#
download eventing "$KNATIVE_EVENTING_VERSION" "${eventing_files[@]}"

# Drop namespace from manifest.
git apply "$root/openshift-knative-operator/hack/001-eventing-namespace-deletion.patch"

# Extra ClusterRole for downstream, so that users can get the CMs of knative-eventing
# TODO: propose to upstream
git apply "$root/openshift-knative-operator/hack/002-openshift-eventing-role.patch"

# For SRVKE-934 we disable HPA:
git apply "$root/openshift-knative-operator/hack/005-disable-hpa.patch"

# TODO: Remove this once upstream fixed https://github.com/knative/operator/issues/376.
# This is the eventing counterpart of SRVKS-670.
git apply "$root/openshift-knative-operator/hack/006-eventing-pdb.patch"

# TODO: Remove this when we update to upstream 1.1 (for SO 1.22)
git apply "$root/openshift-knative-operator/hack/009-sequence.patch"
