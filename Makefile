# Useful for local development
dev:
	./hack/dev.sh
.PHONY: dev

# General purpose targets
images:
	./hack/images.sh $(DOCKER_REPO_OVERRIDE)
.PHONY: images

install:
	./hack/install.sh
.PHONY: install

teardown:
	./hack/teardown.sh
.PHONY: teardown

# Test targets for CI operator
test-unit:
	go test ./serving/ingress/...
.PHONY: test-e2e

test-e2e:
	./test/e2e-tests.sh
.PHONY: test-e2e

# TODO: do we need eventing as well?
test-upgrade:
	./test/upgrade-tests.sh
.PHONY: test-upgrade

# TODO: does this target needs modification for eventing?
# Generates a ci-operator configuration for a specific branch.
generate-ci-config:
	./openshift/ci-operator/generate-ci-config.sh $(BRANCH) > ci-operator-config.yaml
.PHONY: generate-ci-config
