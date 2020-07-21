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

teardown:
	./hack/teardown.sh

# Test targets for CI operator.
test-unit:
	go test ./knative-operator/...
	go test ./serving/ingress/...

# Run only tests from the current repo.
test-operator:
	./test/operator-tests.sh

# Run third-party E2E (Serving, Eventing).
test-e2e:
	TEST_ALL=false ./test/e2e-tests.sh

# Run all tests. Used by periodic CI jobs.
test-all:
	TEST_ALL=true ./test/e2e-tests.sh

test-upgrade:
	./test/upgrade-tests.sh

# Generates a ci-operator configuration for a specific branch.
generate-ci-config:
	./openshift/ci-operator/generate-ci-config.sh $(BRANCH) > ci-operator-config.yaml
