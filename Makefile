# Useful for local development
dev:
	./hack/dev.sh

# General purpose targets
images:
	./hack/images.sh $(DOCKER_REPO_OVERRIDE)

install:
	./hack/install.sh

install-operator:
	INSTALL_SERVING="false" INSTALL_EVENTING="false" ./hack/install.sh

install-all: install-strimzi
	INSTALL_KAFKA="true" ./hack/install.sh

install-serving:
	INSTALL_EVENTING="false" ./hack/install.sh

install-eventing:
	INSTALL_SERVING="false" ./hack/install.sh

install-kafka:
	INSTALL_SERVING="false" INSTALL_KAFKA="true" ./hack/install.sh

install-strimzi:
	UNINSTALL_STRIMZI="false" ./hack/strimzi.sh

uninstall-strimzi:
	UNINSTALL_STRIMZI="true" ./hack/strimzi.sh

install-previous:
	INSTALL_PREVIOUS_VERSION="true" ./hack/install.sh

install-mesh:
	UNINSTALL_MESH="false" ./hack/mesh.sh

uninstall-mesh:
	UNINSTALL_MESH="true" ./hack/mesh.sh

install-full-mesh:
	FULL_MESH="true" UNINSTALL_MESH="false" ./hack/mesh.sh

uninstall-full-mesh:
	FULL_MESH="true" UNINSTALL_MESH="true" ./hack/mesh.sh

teardown:
	./hack/teardown.sh

# Test targets for CI operator.
test-unit:
	go test ./knative-operator/...
	go test ./openshift-knative-operator/...
	go test ./serving/ingress/...

# Run only SERVING/EVENTING E2E tests from the current repo.
test-e2e:
	./test/e2e-tests.sh

# Run E2E tests from the current repo for serving+eventing+knativeKafka
test-e2e-with-kafka:
	INSTALL_KAFKA=true TEST_KNATIVE_KAFKA=true ./test/e2e-tests.sh

# Run both unit and E2E tests from the current repo.
test-operator: test-unit test-e2e

# Run upstream E2E tests including upgrades (Serving, Eventing, ...).
test-upstream-e2e:
	UNINSTALL_STRIMZI="false" ./hack/strimzi.sh
	INSTALL_KAFKA=true TEST_KNATIVE_KAFKA=true ./test/upstream-e2e-tests.sh

# Run upstream E2E tests with net-istio and sidecar.
# TODO: Enable upgrade tests once upstream fixed the issue https://github.com/knative/serving/issues/11535.
test-upstream-e2e-mesh:
	UNINSTALL_STRIMZI="false" ./hack/strimzi.sh
	FULL_MESH=true INSTALL_KAFKA=false TEST_KNATIVE_KAFKA=false TEST_KNATIVE_UPGRADE=false ./test/upstream-e2e-tests.sh

# Run upstream E2E tests without upgrades.
test-upstream-e2e-no-upgrade:
	UNINSTALL_STRIMZI="false" ./hack/strimzi.sh
	INSTALL_KAFKA=true TEST_KNATIVE_KAFKA=true TEST_KNATIVE_E2E=true TEST_KNATIVE_UPGRADE=false ./test/upstream-e2e-tests.sh

# Run only upstream upgrade tests.
test-upstream-upgrade:
	UNINSTALL_STRIMZI="false" ./hack/strimzi.sh
	INSTALL_KAFKA=true TEST_KNATIVE_E2E=false TEST_KNATIVE_UPGRADE=true ./test/upstream-e2e-tests.sh

# Alias.
test-upgrade: test-upstream-upgrade

# Run Console UI e2e tests.
test-ui-e2e:
	./test/ui-e2e-tests.sh

# Run all E2E tests. Used by periodic CI jobs.
test-all-e2e: test-e2e test-upstream-e2e test-ui-e2e

# Generates a ci-operator configuration for a specific branch.
generate-ci-config:
	./openshift/ci-operator/generate-ci-config.sh $(BRANCH) > ci-operator-config.yaml

# Generates all files that are templated with release metadata.
release-files:
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
		templates/stopbundle.Dockerfile \
		olm-catalog/serverless-operator/stopbundle.Dockerfile
	./hack/generate/dockerfile.sh \
		templates/test-source-image.Dockerfile \
		openshift/ci-operator/source-image/Dockerfile
	./hack/generate/dockerfile.sh \
		templates/build-image.Dockerfile \
		openshift/ci-operator/build-image/Dockerfile

# Generates all files that can be generated, includes release files, code generation
# and updates vendoring.
generated-files: release-files
	./hack/update-deps.sh
	./hack/update-codegen.sh
	(cd knative-operator && ./hack/update-manifests.sh)
	(cd openshift-knative-operator && ./hack/update-manifests.sh)
	(cd olm-catalog/serverless-operator && ./hack/update-manifests.sh)
	./hack/update-deps.sh

# Runs the lints Github Actions do too, applies fixes where possible.
lint:
	woke
	golangci-lint run
	find . -type f -path './**/*.*sh' -not -path '*vendor*' | xargs -r shellcheck
	operator-sdk bundle validate ./olm-catalog/serverless-operator
	git ls-files | grep -Ev '^(vendor/|.git)' | xargs misspell -error
	prettier --write templates/*.yaml
