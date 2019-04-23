ifndef DEV_MK
DEV_MK:=# Prevent repeated "-include".

include ./make/verbose.mk
include ./make/git.mk

DOCKER_REPO?=quay.io/openshiftio
IMAGE_NAME?=toolchain-operator
REGISTRY_URI=quay.io

TIMESTAMP:=$(shell date +%s)
TAG?=$(GIT_COMMIT_ID_SHORT)-$(TIMESTAMP)
OPENSHIFT_VERSION?=4

DEPLOY_DIR:=deploy

.PHONY: push-operator-image
## Push the operator container image to a container registry
push-operator-image: build-operator-image
	@docker login -u $(QUAY_USERNAME) -p $(QUAY_PASSWORD) $(REGISTRY_URI)
	docker push $(DOCKER_REPO)/$(IMAGE_NAME):$(TAG)

.PHONY: create-resources
create-resources:
	@echo "Logging using system:admin..."
	@oc login -u system:admin
	@echo "Creating sub resources..."
	@echo "Creating CRDs..."
	@oc create -f $(DEPLOY_DIR)/crds/codeready_v1alpha1_toolchainenabler_crd.yaml
	@echo "Creating Namespace"
	@oc create -f $(DEPLOY_DIR)/namespace.yaml
	@echo "oc project codeready-toolchain"
	@oc project codeready-toolchain
	@echo "Creating Service Account"
	@oc create -f $(DEPLOY_DIR)/service_account.yaml
	@echo "Creating Role"
	@oc create -f $(DEPLOY_DIR)/role.yaml
	@echo "Creating RoleBinding"
	@oc create -f $(DEPLOY_DIR)/role_binding.yaml
	@echo "Creating Cluster Role"
	@oc create -f $(DEPLOY_DIR)/cluster_role.yaml
	@echo "Creating ClusterRoleBinding"
	@oc create -f $(DEPLOY_DIR)/cluster_role_binding.yaml
	@echo "Creating Secret"
	@oc create -f $(DEPLOY_DIR)/operator.config.yaml
	@echo "Creating dsaas-cluster-admin ClusterRole"
	@oc create -f $(DEPLOY_DIR)/olm-catalog/manifests/0.0.2/dsaas-cluster-admin.ClusterRole.yaml
	@echo "Creating online-registration ClusterRole"
	@oc create -f $(DEPLOY_DIR)/olm-catalog/manifests/0.0.2/online-registration.ClusterRole.yaml

.PHONY: create-cr
create-cr:
	@echo "Creating Custom Resource..."
	@oc create -f $(DEPLOY_DIR)/crds/codeready_v1alpha1_toolchainenabler_cr.yaml

.PHONY: build-operator-image
build-operator-image:
	docker build -t $(DOCKER_REPO)/$(IMAGE_NAME):$(TAG) -f Dockerfile.dev .
	docker tag $(DOCKER_REPO)/$(IMAGE_NAME):$(TAG) $(DOCKER_REPO)/$(IMAGE_NAME):test

.PHONY: deploy-operator-only
deploy-operator-only:
	@echo "Creating Deployment for Operator"
	@cat minishift/operator.yaml | sed s/\:dev/:$(TAG)/ | oc create -f -

.PHONY: clean-all
clean-all:  clean-operator clean-resources

.PHONY: clean-operator
clean-operator:
	@echo "Deleting Deployment for Operator"
	@cat minishift/operator.yaml | sed s/\:dev/:$(TAG)/ | oc delete -f - || true

.PHONY: clean-resources
clean-resources:
	@echo "Deleting sub resources..."
	@echo "Deleting RoleBinding"
	@oc delete -f $(DEPLOY_DIR)/role_binding.yaml || true
	@echo "Deleting ClusterRoleBinding"
	@oc delete -f $(DEPLOY_DIR)/cluster_role_binding.yaml || true
	@echo "Deleting Role"
	@oc delete -f $(DEPLOY_DIR)/role.yaml || true
	@echo "Deleting ClusterRole"
	@oc delete -f $(DEPLOY_DIR)/cluster_role.yaml || true
	@echo "Deleting Service Account"
	@oc delete -f $(DEPLOY_DIR)/service_account.yaml || true
	@echo "Deleting Custom Resources..."
	@oc delete -f $(DEPLOY_DIR)/crds/codeready_v1alpha1_toolchainenabler_cr.yaml || true
	@echo "Deleting Namespace"
	@oc delete -f $(DEPLOY_DIR)/namespace.yaml || true
	@echo "Deleting Custom Resource Definitions..."
	@oc delete -f $(DEPLOY_DIR)/crds/codeready_v1alpha1_toolchainenabler_crd.yaml || true
	@echo "Deleting OAuthClient 'codeready-toolchain'"
	@oc delete oauthclient codeready-toolchain || true
	@echo "Deleting Secret"
	@oc delete -f $(DEPLOY_DIR)/operator.config.yaml || true
	@echo "Deleting dsaas-cluster-admin ClusterRole"
	@oc delete -f $(DEPLOY_DIR)/olm-catalog/manifests/0.0.2/dsaas-cluster-admin.ClusterRole.yaml || true
	@echo "Deleting online-registration ClusterRole"
	@oc delete -f $(DEPLOY_DIR)/olm-catalog/manifests/0.0.2/online-registration.ClusterRole.yaml || true
	@echo "Deleting online-registration ClusterRoleBinding"
	@oc delete clusterrolebinding online-registration || true
	@echo "Deleting online-registration sa from openshift-infra namespace"
	@oc delete sa online-registration -n openshift-infra || true

.PHONY: deploy-operator
deploy-operator: build build-operator-image deploy-operator-only

.PHONY: minishift-start
minishift-start:
	minishift start --cpus 4 --memory 8GB
	-eval `minishift docker-env` && oc login -u system:admin

.PHONY: deploy-all
deploy-all: clean-resources create-resources deploy-operator create-cr

endif
