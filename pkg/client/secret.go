package client

import (
	"context"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

// CreateSecret creates the Secret.
func (c *clientImpl) CreateSecret(s *v1.Secret) error {
	return c.Client.Create(context.Background(), s)

}

// GetSecret returns the existing Secret.
func (c *clientImpl) GetSecret(namespace, name string) (*v1.Secret, error) {
	s := &v1.Secret{}
	if err := c.Client.Get(context.Background(), types.NamespacedName{Namespace: namespace, Name: name}, s); err != nil {
		return nil, err
	}
	return s, nil
}
