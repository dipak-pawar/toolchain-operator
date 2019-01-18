package client


import (
	"k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)


type ClientInterface interface {
	KubernetesInterface() kubernetes.Interface
	Secret
	ServiceAccount
	ClusterRoleBinding
}

// Secret contains methods for manipulating Secrets
type Secret interface {
	CreateSecret(*v1.Secret) (*v1.Secret, error)
	GetSecret(namespace, name string) (*v1.Secret, error)
	DeleteSecret(namespace, name string, options *metav1.DeleteOptions) error
}

// ServiceAccount contains methods for manipulating ServiceAccounts.
type ServiceAccount interface {
	CreateServiceAccount(*v1.ServiceAccount) (*v1.ServiceAccount, error)
	GetServiceAccount(namespace, name string) (*v1.ServiceAccount, error)
	DeleteServiceAccount(namespace, name string, options *metav1.DeleteOptions) error
}

// ClusterRoleBinding contains methods for manipulating ClusterRoleBindings.
type ClusterRoleBinding interface {

	CreateClusterRoleBinding(*rbacv1.ClusterRoleBinding) (*rbacv1.ClusterRoleBinding, error)
	GetClusterRoleBinding(name string) (*rbacv1.ClusterRoleBinding, error)
	DeleteClusterRoleBinding(name string, options *metav1.DeleteOptions) error
}

// Interface assertion.
var _ ClientInterface = &Client{}

// Client is a kubernetes client that can talk to the API server.
type Client struct {
	kubernetes.Interface
}

// NewClientFromConfig creates a kubernetes client
func NewClientFromConfig(config *rest.Config) ClientInterface {
	return &Client{kubernetes.NewForConfigOrDie(config)}
}

// NewClient creates a kubernetes client
func NewClient(k8sClient kubernetes.Interface) ClientInterface {
	return &Client{k8sClient}
}

// KubernetesInterface returns the Kubernetes interface.
func (c *Client) KubernetesInterface() kubernetes.Interface {
	return c.Interface
}
