package online_registration

import (
	"fmt"
	"github.com/fabric8-services/toolchain-operator/pkg/client"
	"github.com/fabric8-services/toolchain-operator/pkg/config"
	errs "github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var log = logf.Log.WithName("online_registration_resource_creator")

var ServiceAccountName = "online-registration"
var Namespace = "openshift-infra"
var ClusterRoleBindingName = "online-registration"

var serviceAccount = corev1.ServiceAccount{
	ObjectMeta: metav1.ObjectMeta{
		Name:      ServiceAccountName,
		Namespace: Namespace,
	},
}

var clusterRoleBinding = rbacv1.ClusterRoleBinding{
	ObjectMeta: metav1.ObjectMeta{
		Name: ClusterRoleBindingName,
	},
	Subjects: []rbacv1.Subject{
		{
			Kind: "ServiceAccount",
			Name: ServiceAccountName,
		},
	},
	RoleRef: rbacv1.RoleRef{
		APIGroup: "rbac.authorization.k8s.io",
		Kind:     "ClusterRole",
		Name:     "online-registration",
	},
}

type resourceCreator struct {
	client client.Client
}

func EnsureResources(c client.Client) error {
	r := resourceCreator{client: c}
	//
	if err := r.createServiceAccount(); err != nil {
		return err
	}

	if err := r.createClusterRoleBinding(); err != nil {
		return err
	}

	return nil
}

func (r resourceCreator) createServiceAccount() error {
	if _, err := r.client.GetServiceAccount(Namespace, ServiceAccountName); err != nil {
		if errors.IsNotFound(err) {
			log.Info("creating a new service account ", "namespace", Namespace, "name", ServiceAccountName)
			if err := r.client.CreateServiceAccount(&serviceAccount); err != nil {
				return err
			}
			log.Info(fmt.Sprintf("service account %s created successfully", config.SAName))
			return nil
		}
		return errs.Wrapf(err, "failed to get service account %s", config.SAName)
	}
	log.Info(fmt.Sprintf("service account %s already exists", config.SAName))
	return nil
}

func (r resourceCreator) createClusterRoleBinding() error {
	if _, err := r.client.GetClusterRoleBinding(ClusterRoleBindingName); err != nil {
		if errors.IsNotFound(err) {
			log.Info("adding online-registration cluster role to", "service account", ServiceAccountName)
			if err := r.client.CreateClusterRoleBinding(&clusterRoleBinding); err != nil {
				return err
			}

			log.Info(fmt.Sprintf("clusterrolebinding %s created successfully", ClusterRoleBindingName))
			return nil
		}
		return errs.Wrapf(err, "failed to get clusterrolebinding %s", ClusterRoleBindingName)
	}

	log.Info(fmt.Sprintf("clusterrolebinding %s already exists", ClusterRoleBindingName))
	return nil
}
