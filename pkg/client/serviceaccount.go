package client

import (
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreateServiceAccount creates the serviceAccount.
func (c *Client) CreateServiceAccount(ig *v1.ServiceAccount) (*v1.ServiceAccount, error) {
	return c.CoreV1().ServiceAccounts(ig.GetNamespace()).Create(ig)
}

// GetServiceAccount returns the existing serviceAccount.
func (c *Client) GetServiceAccount(namespace, name string) (*v1.ServiceAccount, error) {
	return c.CoreV1().ServiceAccounts(namespace).Get(name, metav1.GetOptions{})
}

