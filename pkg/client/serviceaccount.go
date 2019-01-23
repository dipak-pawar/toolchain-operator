package client

import (
	"context"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

// CreateServiceAccount creates the serviceAccount.
func (c *Client) CreateServiceAccount(sa *v1.ServiceAccount) error {
	return c.Create(context.Background(), sa)
}

// GetServiceAccount returns the existing serviceAccount.
func (c *Client) GetServiceAccount(namespace, name string) (*v1.ServiceAccount, error) {
	sa := &v1.ServiceAccount{}
	if err := c.Get(context.Background(), types.NamespacedName{Namespace: namespace, Name: name}, sa); err != nil {
		return nil, err
	}
	return sa, nil
}
