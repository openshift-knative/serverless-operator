#This makefile is used by ci-operator

test-unit:
	go test ./serving/ingress/...
.PHONY: test-e2e

test-e2e:
	./test/e2e-tests.sh
.PHONY: test-e2e

install:
	# Do nothing right now. Required by ci-operator.
.PHONY: install

# Generates a ci-operator configuration for a specific branch.
generate-ci-config:
	./openshift/ci-operator/generate-ci-config.sh $(BRANCH) > ci-operator-config.yaml
.PHONY: generate-ci-config