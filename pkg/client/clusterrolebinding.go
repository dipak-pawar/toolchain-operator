package client

import (
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreateRoleBinding creates the roleBinding.
func (c *Client) CreateClusterRoleBinding(ig *rbacv1.ClusterRoleBinding) (*rbacv1.ClusterRoleBinding, error) {
	return c.RbacV1().ClusterRoleBindings().Create(ig)
}

// GetRoleBinding returns the existing roleBinding.
func (c *Client) GetClusterRoleBinding(name string) (*rbacv1.ClusterRoleBinding, error) {
	return c.RbacV1().ClusterRoleBindings().Get(name, metav1.GetOptions{})
}
