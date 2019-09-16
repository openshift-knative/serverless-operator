#This makefile is used by ci-operator

test-e2e:
	./test/e2e-tests.sh
.PHONY: test-e2e
