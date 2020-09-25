# Useful for local development
dev:
	./hack/dev.sh

# General purpose targets
images:
	./hack/images.sh $(DOCKER_REPO_OVERRIDE)

install:
	./hack/install.sh

install-previous:
	INSTALL_PREVIOUS_VERSION="true" ./hack/install.sh

install-mesh:
	UNINSTALL_MESH="false" ./hack/mesh.sh

uninstall-mesh:
	UNINSTALL_MESH="true" ./hack/mesh.sh

teardown:
	./hack/teardown.sh

# Test targets for CI operator.
test-unit:
	go test ./knative-operator/...
	go test ./serving/ingress/...

# Run only E2E tests from the current repo.
test-e2e:
	./test/e2e-tests.sh

# Run both unit and E2E tests from the current repo.
test-operator: test-unit test-e2e

# Run upstream E2E tests including upgrades (Serving, Eventing, ...).
test-upstream-e2e:
	./test/upstream-e2e-tests.sh

# Run upstream E2E tests without upgrades.
test-upstream-e2e-no-upgrade:
	TEST_KNATIVE_E2E=true TEST_KNATIVE_UPGRADE=false ./test/upstream-e2e-tests.sh

# Run only upstream upgrade tests.
test-upstream-upgrade:
	TEST_KNATIVE_E2E=false TEST_KNATIVE_UPGRADE=true ./test/upstream-e2e-tests.sh

# Alias.
test-upgrade: test-upstream-upgrade

# Run all E2E tests. Used by periodic CI jobs.
test-all-e2e: test-e2e test-upstream-e2e

# Generates a ci-operator configuration for a specific branch.
generate-ci-config:
	./openshift/ci-operator/generate-ci-config.sh $(BRANCH) > ci-operator-config.yaml

csv:
	./olm-catalog/serverless-operator/generate_csv.sh olm-catalog/serverless-operator/csv.template.yaml olm-catalog/serverless-operator/manifests/serverless-operator.clusterserviceversion.yaml
