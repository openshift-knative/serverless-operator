# Useful for local development
publish-images:
	./hack/publish.sh $(DOCKER_REPO_OVERRIDE)
.PHONY: publish-images

# Test targets for CI operator
test-unit:
	go test ./serving/ingress/...
.PHONY: test-e2e

test-e2e:
	./test/e2e-tests.sh
.PHONY: test-e2e

test-upgrade:
	./test/upgrade-tests.sh
.PHONY: test-upgrade

# Generates a ci-operator configuration for a specific branch.
generate-ci-config:
	./openshift/ci-operator/generate-ci-config.sh $(BRANCH) > ci-operator-config.yaml
.PHONY: generate-ci-config