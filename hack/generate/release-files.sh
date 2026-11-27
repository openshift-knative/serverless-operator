#!/usr/bin/env bash

# Generates all files that are templated with release metadata.
#
# All images are resolved once here, with registry lookups done in parallel, and exported. The
# generators run as child processes and inherit the resolved images (image variables are only
# resolved when not set already), so none of them has to look images up again.

set -Eeuo pipefail

root="$(dirname "$(dirname "$(dirname "$(realpath "${BASH_SOURCE[0]}")")")")"
cd "${root}"

# shellcheck disable=SC1091,SC1090
source "${root}/hack/lib/common.bash"
# shellcheck disable=SC1091,SC1090
source "${root}/hack/lib/metadata.bash"
# shellcheck disable=SC1091,SC1090
source "${root}/hack/lib/images.bash"

images.init_inspect_cache
mapfile -t resolvers < <(images.csv_resolvers)
images.prefetch "${resolvers[@]}"
for resolver in "${resolvers[@]}"; do
  "${resolver}"
done

./hack/generate/csv.sh \
  templates/csv.yaml \
  olm-catalog/serverless-operator/manifests/serverless-operator.clusterserviceversion.yaml
./hack/generate/annotations.sh \
  templates/annotations.yaml \
  olm-catalog/serverless-operator/metadata/annotations.yaml
./hack/generate/dockerfile.sh \
  templates/main.Dockerfile \
  olm-catalog/serverless-operator/Dockerfile
./hack/generate/dockerfile.sh \
  templates/test-source-image.Dockerfile \
  openshift/ci-operator/source-image/Dockerfile
./hack/generate/dockerfile.sh \
  templates/build-image.Dockerfile \
  openshift/ci-operator/build-image/Dockerfile
./hack/generate/index.sh \
  olm-catalog/serverless-operator-index/configs/index.yaml
./hack/generate/dockerfile.sh \
  templates/catalog.Dockerfile \
  olm-catalog/serverless-operator-index
./hack/generate/dockerfile.sh \
  templates/index.Dockerfile \
  olm-catalog/serverless-operator-index/Dockerfile
./hack/generate/quickstart.sh \
  templates/serverless-application-quickstart.yaml \
  knative-operator/deploy/resources/quickstart/serverless-application-quickstart.yaml
./hack/generate/images-rekt.sh \
  templates/images-rekt.yaml \
  test/images-rekt.yaml
./hack/generate/mesh-auth-policies.sh \
  tenant-1,tenant-2,serving-tests,serverless-tests,eventing-e2e0,eventing-e2e1,eventing-e2e2,eventing-e2e3,eventing-e2e4
./hack/generate/metadata-webhook.sh
