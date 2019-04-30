ifndef TEST_MK
TEST_MK:=# Prevent repeated "-include".
UNAME_S := $(shell uname -s)

include ./make/verbose.mk
include ./make/out.mk

.PHONY: test
## Runs Go package tests and stops when the first one fails
test: ./vendor
	$(Q)go test -vet off ${V_FLAG} $(shell go list ./... | grep -v /test/e2e) -failfast

.PHONY: test-coverage
## Runs Go package tests and produces coverage information
test-coverage: ./out/cover.out

.PHONY: test-coverage-html
## Gather (if needed) coverage information and show it in your browser
test-coverage-html: ./vendor ./out/cover.out
	$(Q)go tool cover -html=./out/cover.out

./out/cover.out: ./vendor
	$(Q)go test ${V_FLAG} -race $(shell go list ./... | grep -v /test/e2e) -failfast -coverprofile=cover.out -covermode=atomic -outputdir=./out

.PHONY: get-test-namespace
get-test-namespace: ./out/test-namespace
	$(eval TEST_NAMESPACE := $(shell cat ./out/test-namespace))

./out/test-namespace:
	@echo -n "test-namespace-$(shell uuidgen | tr '[:upper:]' '[:lower:]')" > ./out/test-namespace

.PHONY: test-e2e
## Runs the e2e tests locally
test-e2e: ./vendor e2e-setup create-resources deploy-operator
	$(info Running E2E test: $@)
ifeq ($(OPENSHIFT_VERSION),3)
	$(Q)oc login -u system:admin
endif
	$(Q)operator-sdk test local ./test/e2e --no-setup --debug --namespace $(NAMESPACE) --go-test-flags "-v -timeout=15m"

.PHONY: e2e-setup
e2e-setup: e2e-cleanup 
	$(Q)oc new-project $(NAMESPACE)

.PHONY: e2e-cleanup
e2e-cleanup: get-test-namespace
	$(Q)oc delete project $(NAMESPACE) --timeout=10s --wait || true

endif
