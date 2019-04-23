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
test-e2e: ./vendor e2e-setup
	$(info Running E2E test: $@)
ifeq ($(OPENSHIFT_VERSION),3)
	$(Q)oc login -u system:admin
endif
	$(Q)operator-sdk test local ./test/e2e --namespace $(TEST_NAMESPACE) --up-local --global-manifest deploy/test/global-manifests.yaml --namespaced-manifest deploy/test/namespace-manifests.yaml --go-test-flags "-v -timeout=15m"

.PHONY: test-e2e
## Runs the e2e tests and WITHOUT producing coverage files for each package.
test-e2e: build build-image e2e-setup
	$(call log-info,"Running E2E test: $@")
	go test ./test/e2e/... -root=$(PWD) -kubeconfig=$(HOME)/.kube/config -globalMan deploy/test/global-manifests.yaml -namespacedMan deploy/test/namespace-manifests.yaml -v -parallel=1 -singleNamespace

 .PHONY: e2e-setup
e2e-setup:  e2e-cleanup
	oc new-project toolchain-e2e-test || true

.PHONY: e2e-setup
e2e-setup: e2e-cleanup 
	$(Q)oc new-project toolchain-e2e-test

.PHONY: e2e-cleanup
e2e-cleanup: get-test-namespace
	$(Q)-oc delete project toolchain-e2e-test --timeout=10s --wait

endif
