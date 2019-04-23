#===============================================================================
# Testing has become a rather big and interconnected topic and that's why it
# has arrived in it's own file.
#
# We have to types of tests available:
#
#  1. unit tests
#
# The unit tests can be executed fairly simply be running `go test`
#
# Usage
# -----
# If you want to run the unit tests, type
#
#     $ make test-unit
#
#
# To run all tests, type
#
#     $ make test-all
#
# To output unit-test coverage profile information for each function, type
#
#     $ make coverage-unit
#
# To generate unit-test HTML representation of coverage profile (opens a browser), type
#
#     $ make coverage-unit-html
#
#
# To output all coverage profile information for each function, type
#
#     $ make coverage-all
#
# Artifacts and coverage modes
# ----------------------------
# Each package generates coverage outputs under tmp/coverage/$(PACKAGE) where
# $(PACKAGE) resolves to the Go package. Here's an example of a coverage file
# for the package "github.com/fabric8-services/fabric8-auth/models" with coverage mode
# "set" generated by the unit tests:
#
#   tmp/coverage/github.com/fabric8-services/fabric8-auth/models/coverage.unit.mode-set
#
# For unit-tests all results are combined into this file:
#
#   tmp/coverage.unit.mode-$(COVERAGE_MODE)
#
#
# The overall coverage gets combined into this file:
#
#   tmp/coverage.mode-$(COVERAGE_MODE)
#
# The $(COVERAGE_MODE) in each filename indicates what coverage mode was used.
#
# These are possible coverage modes (see https://blog.golang.org/cover):
#
# 	set: did each statement run? (default)
# 	count: how many times did each statement run?
# 	atomic: like count, but counts precisely in parallel programs
#
# To choose another coverage mode, simply prefix the invovation of `make`:
#
#     $ COVERAGE_MODE=count make test-unit
#===============================================================================

# mode can be: set, count, or atomic
COVERAGE_MODE ?= set

# By default no go test calls will use the -v switch when running tests.
# But if you want you can enable that by setting GO_TEST_VERBOSITY_FLAG=-v
GO_TEST_VERBOSITY_FLAG ?=


# By default reduce the amount of log output from tests
ADMIN_LOG_LEVEL ?= error

# Output directory for coverage information
COV_DIR = $(TMP_PATH)/coverage

# Files that combine package coverages for unit tests separately
COV_PATH_UNIT = $(TMP_PATH)/coverage.unit.mode-$(COVERAGE_MODE)

# File that stores overall coverge for all packages and unit-tests
COV_PATH_OVERALL = $(TMP_PATH)/coverage.mode-$(COVERAGE_MODE)

# This pattern excludes some folders from the coverage calculation (see grep -v)
ALL_PKGS_EXCLUDE_PATTERN = 'vendor\|app\|tool\/cli\|design\|client\|test'

# This pattern excludes some folders from the go code analysis
GOANALYSIS_PKGS_EXCLUDE_PATTERN="vendor|app|client|tool/cli"
GOANALYSIS_DIRS=$(shell go list -f {{.Dir}} ./... | grep -v -E $(GOANALYSIS_PKGS_EXCLUDE_PATTERN))

#-------------------------------------------------------------------------------
# Normal test targets
#
# These test targets are the ones that will be invoked from the outside. If
# they are called and the artifacts already exist, then the artifacts will
# first be cleaned and recreated. This ensures that the tests are always
# executed.
#-------------------------------------------------------------------------------

.PHONY: test-all
## Runs the test-unit targets.
test-all: prebuild-check test-unit test-e2e

.PHONY: test-unit-with-coverage
# Runs the unit tests and produces coverage files for each package.
test-unit-with-coverage: prebuild-check clean-coverage-unit $(COV_PATH_UNIT)

.PHONY: test-unit
## Runs the unit tests and WITHOUT producing coverage files for each package.
test-unit: prebuild-check $(SOURCES)
	$(call log-info,"Running test: $@")
	$(eval TEST_PACKAGES:=$(shell go list ./... | grep -v $(ALL_PKGS_EXCLUDE_PATTERN)))
	ADMIN_LOG_LEVEL=$(ADMIN_LOG_LEVEL) go test -vet off $(GO_TEST_VERBOSITY_FLAG) $(TEST_PACKAGES)

.PHONY: test-e2e
## Runs the e2e tests and WITHOUT producing coverage files for each package.
test-e2e: build build-image e2e-setup create-olm-resources
	$(call log-info,"Running E2E test: $@")
	go test ./test/e2e/... -root=$(PWD) -kubeconfig=$(HOME)/.kube/config -globalMan deploy/test/global-manifests.yaml -namespacedMan deploy/test/namespace-manifests.yaml -v -parallel=1 -singleNamespace

.PHONY: e2e-setup
e2e-setup:  e2e-cleanup
	oc new-project toolchain-e2e-test || true

## this is temporary workaround till until we have olm integration setup for testing is done
.PHONY: create-olm-resources
create-olm-resources:
	oc apply -f deploy/olm-catalog/manifests/0.0.2/dsaas-cluster-admin.ClusterRole.yaml
	oc apply -f deploy/olm-catalog/manifests/0.0.2/online-registration.ClusterRole.yaml

.PHONY: e2e-cleanup
e2e-cleanup:
	oc login -u system:admin
	oc delete oauthclient codeready-toolchain || true
	oc delete clusterrolebinding toolchain-enabler || true
	oc delete clusterrole toolchain-enabler || true
	oc delete project toolchain-e2e-test || true
	oc delete -f deploy/olm-catalog/manifests/0.0.2/dsaas-cluster-admin.ClusterRole.yaml || true
	oc delete -f deploy/olm-catalog/manifests/0.0.2/online-registration.ClusterRole.yaml || true

#-------------------------------------------------------------------------------
# Inspect coverage of unit tests, integration tests in either pure
# console mode or in a browser (*-html).
#
# If the test coverage files to be evaluated already exist, then no new
# tests are executed. If they don't exist, we first run the tests.
#-------------------------------------------------------------------------------

# Prints the total coverage of a given package.
# The total coverage is printed as the last argument in the
# output of "go tool cover". If the requested test name (first argument)
# Is *, then unit, integration  tests will be combined
define print-package-coverage
$(eval TEST_NAME:=$(1))
$(eval PACKAGE_NAME:=$(2))
$(eval COV_FILE:="$(COV_DIR)/$(PACKAGE_NAME)/coverage.$(TEST_NAME).mode-$(COVERAGE_MODE)")
 @if [ "$(TEST_NAME)" == "*" ]; then \
  UNIT_FILE=`echo $(COV_FILE) | sed 's/*/unit/'`; \
  INTEGRATON_FILE=`echo $(COV_FILE) | sed 's/*/integration/'`; \
  COV_FILE=`echo $(COV_FILE) | sed 's/*/combined/'`; \
	if [ ! -e $${UNIT_FILE} ]; then \
		COV_FILE=$${INTEGRATION_FILE}; \
	else \
		if [ ! -e $${INTEGRATION_FILE} ]; then \
			COV_FILE=$${UNIT_FILE}; \
		else \
			$(GOCOVMERGE_BIN) $${UNIT_FILE} $${INTEGRATION_FILE} > $${COV_FILE}; \
		fi; \
	fi; \
else \
	COV_FILE=$(COV_FILE); \
fi; \
if [ -e "$${COV_FILE}" ]; then \
	VAL=`go tool cover -func=$${COV_FILE} \
		| grep '^total:' \
		| grep '\S\+$$' -o \
		| sed 's/%//'`; \
	printf "%-80s %#5.2f%%\n" "$(PACKAGE_NAME)" "$${VAL}"; \
else \
	printf "%-80s %6s\n" "$(PACKAGE_NAME)" "n/a"; \
fi
endef

# Iterates over every package and prints its test coverage
# for a given test name ("unit").
define package-coverage
$(eval TEST_NAME:=$(1))
@printf "\n\nPackage coverage:\n"
$(eval TEST_PACKAGES:=$(shell go list ./... | grep -v $(ALL_PKGS_EXCLUDE_PATTERN)))
$(foreach package, $(TEST_PACKAGES), $(call print-package-coverage,$(TEST_NAME),$(package)))
endef

$(COV_PATH_OVERALL): $(GOCOVMERGE_BIN)
	@$(GOCOVMERGE_BIN) $(COV_PATH_UNIT) > $(COV_PATH_OVERALL)

# Console coverage output:

# First parameter: file to do in-place replacement with.
define cleanup-coverage-file
@sed -i '/.*\/sqlbindata\.go.*/d' $(1)
@sed -i '/.*\/confbindata\.go.*/d' $(1)
endef

.PHONY: coverage-unit
# Output coverage profile information for each function (only based on unit-tests).
# Re-runs unit-tests if coverage information is outdated.
coverage-unit: prebuild-check $(COV_PATH_UNIT)
	$(call cleanup-coverage-file,$(COV_PATH_UNIT))
	@go tool cover -func=$(COV_PATH_UNIT)
	$(call package-coverage,unit)


.PHONY: coverage-all
# Output coverage profile information for each function.
# Re-runs unit- and integration-tests if coverage information is outdated.
coverage-all: prebuild-check clean-coverage-overall $(COV_PATH_OVERALL)
	$(call cleanup-coverage-file,$(COV_PATH_OVERALL))
	@go tool cover -func=$(COV_PATH_OVERALL)
	$(call package-coverage,*)

# HTML coverage output:

.PHONY: coverage-unit-html
# Generate HTML representation (and show in browser) of coverage profile (based on unit tests).
# Re-runs unit tests if coverage information is outdated.
coverage-unit-html: prebuild-check $(COV_PATH_UNIT)
	$(call cleanup-coverage-file,$(COV_PATH_UNIT))
	@go tool cover -html=$(COV_PATH_UNIT)

.PHONY: coverage-all-html
# Output coverage profile information for each function.
# Re-runs unit if coverage information is outdated.
coverage-all-html: prebuild-check clean-coverage-overall $(COV_PATH_OVERALL)
	$(call cleanup-coverage-file,$(COV_PATH_OVERALL))
	@go tool cover -html=$(COV_PATH_OVERALL)

# Experimental:

.PHONY: gocov-unit-annotate
# (EXPERIMENTAL) Show actual code and how it is covered with unit tests.
#                This target only runs the tests if the coverage file does exist.
gocov-unit-annotate: prebuild-check $(GOCOV_BIN) $(COV_PATH_UNIT)
	$(call cleanup-coverage-file,$(COV_PATH_UNIT))
	@$(GOCOV_BIN) convert $(COV_PATH_UNIT) | $(GOCOV_BIN) annotate -

.PHONY: .gocov-unit-report
.gocov-unit-report: prebuild-check $(GOCOV_BIN) $(COV_PATH_UNIT)
	$(call cleanup-coverage-file,$(COV_PATH_UNIT))
	@$(GOCOV_BIN) convert $(COV_PATH_UNIT) | $(GOCOV_BIN) report


#-------------------------------------------------------------------------------
# Test artifacts are coverage files for unit and integration tests.
#-------------------------------------------------------------------------------

# The test-package function executes tests for a package and saves the collected
# coverage output to a directory. After storing the coverage information it is
# also appended to a file of choice (without the "mode"-line)
#
# Parameters:
#  1. Test name (e.g. "unit" or "integration")
#  2. package name "github.com/fabric8-services/fabric8-auth/model"
#  3. File in which to combine the output
#  4. Path to file in which to store names of packages that failed testing
#  5. Environment variable (in the form VAR=VALUE) to be specified for running
#     the test. For multiple environment variables, pass "VAR1=VAL1 VAR2=VAL2".
define test-package
$(eval TEST_NAME := $(1))
$(eval PACKAGE_NAME := $(2))
$(eval COMBINED_OUT_FILE := $(3))
$(eval ERRORS_FILE := $(4))
$(eval ENV_VAR := $(5))
$(eval ALL_PKGS_COMMA_SEPARATED := $(6))
@mkdir -p $(COV_DIR)/$(PACKAGE_NAME);
$(eval COV_OUT_FILE := $(COV_DIR)/$(PACKAGE_NAME)/coverage.$(TEST_NAME).mode-$(COVERAGE_MODE))
@$(ENV_VAR) ADMIN_LOG_LEVEL=$(ADMIN_LOG_LEVEL) \
	go test -vet off $(PACKAGE_NAME) \
		$(GO_TEST_VERBOSITY_FLAG) \
		-coverprofile $(COV_OUT_FILE) \
		-coverpkg $(ALL_PKGS_COMMA_SEPARATED) \
		-covermode=$(COVERAGE_MODE) \
		-timeout 10m \
		$(EXTRA_TEST_PARAMS) \
	|| echo $(PACKAGE_NAME) >> $(ERRORS_FILE)

@if [ -e "$(COV_OUT_FILE)" ]; then \
	if [ ! -e "$(COMBINED_OUT_FILE)" ]; then \
		cp $(COV_OUT_FILE) $(COMBINED_OUT_FILE); \
	else \
		cp $(COMBINED_OUT_FILE) $(COMBINED_OUT_FILE).tmp; \
		$(GOCOVMERGE_BIN) $(COV_OUT_FILE) $(COMBINED_OUT_FILE).tmp > $(COMBINED_OUT_FILE); \
	fi \
fi
endef

# Exits the makefile with an error if the file (first parameter) exists.
# Before exiting, the contents of the passed file is printed.
define check-test-results
$(eval ERRORS_FILE := $(1))
@if [ -e "$(ERRORS_FILE)" ]; then \
echo ""; \
echo "ERROR: The following packages did not pass the tests:"; \
echo "-----------------------------------------------------"; \
cat $(ERRORS_FILE); \
echo "-----------------------------------------------------"; \
echo ""; \
exit 1; \
fi
endef

# NOTE: We don't have prebuild-check as a dependency here because it would cause
#       the recipe to be always executed.
$(COV_PATH_UNIT): $(SOURCES) $(GOCOVMERGE_BIN)
	$(eval TEST_NAME := unit)
	$(eval ERRORS_FILE := $(TMP_PATH)/errors.$(TEST_NAME))
	$(call log-info,"Running test: $(TEST_NAME)")
	@mkdir -p $(COV_DIR)
	@echo "mode: $(COVERAGE_MODE)" > $(COV_PATH_UNIT)
	@-rm -f $(ERRORS_FILE)
	$(eval TEST_PACKAGES:=$(shell go list ./... | grep -v $(ALL_PKGS_EXCLUDE_PATTERN)))
	$(eval ALL_PKGS_COMMA_SEPARATED:=$(shell echo $(TEST_PACKAGES)  | tr ' ' ,))
	$(foreach package, $(TEST_PACKAGES), $(call test-package,$(TEST_NAME),$(package),$(COV_PATH_UNIT),$(ERRORS_FILE),,$(ALL_PKGS_COMMA_SEPARATED)))
	$(call check-test-results,$(ERRORS_FILE))

#-------------------------------------------------------------------------------
# Additional tools to build
#-------------------------------------------------------------------------------

$(GOCOV_BIN): prebuild-check
	@cd $(VENDOR_DIR)/github.com/axw/gocov/gocov/ && go build

$(GOCOVMERGE_BIN): prebuild-check
	@cd $(VENDOR_DIR)/github.com/wadey/gocovmerge && go build


#-------------------------------------------------------------------------------
# Clean targets
#-------------------------------------------------------------------------------

CLEAN_TARGETS += clean-coverage
.PHONY: clean-coverage
# Removes all coverage files
clean-coverage: clean-coverage-unit clean-coverage-overall
	-@rm -rf $(COV_DIR)

CLEAN_TARGETS += clean-coverage-overall
.PHONY: clean-coverage-overall
# Removes overall coverage file
clean-coverage-overall:
	-@rm -f $(COV_PATH_OVERALL)

CLEAN_TARGETS += clean-coverage-unit
.PHONY: clean-coverage-unit
# Removes unit test coverage file
clean-coverage-unit:
	-@rm -f $(COV_PATH_UNIT)
