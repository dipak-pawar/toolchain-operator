package client

import (
	oauthv1 "github.com/openshift/api/oauth/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreateOauthClient creates the OauthClient.
func (c *clientImpl) CreateOAuthClient(oc *oauthv1.OAuthClient) error {
	if _, err := c.oauthClient.Create(oc); err != nil {
		return err
	}
	return nil
}

// GetOauthClient returns the existing OAuthClient.
func (c *clientImpl) GetOAuthClient(name string) (*oauthv1.OAuthClient, error) {
	oc, err := c.oauthClient.Get(name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return oc, nil
}
