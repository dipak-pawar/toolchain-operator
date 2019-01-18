package client

import (
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreateSecret creates the Secret.
func (c *Client) CreateSecret(ig *v1.Secret) (*v1.Secret, error) {
	return c.CoreV1().Secrets(ig.GetNamespace()).Create(ig)
}

// GetSecret returns the existing Secret.
func (c *Client) GetSecret(namespace, name string) (*v1.Secret, error) {
	return c.CoreV1().Secrets(namespace).Get(name, metav1.GetOptions{})
}


