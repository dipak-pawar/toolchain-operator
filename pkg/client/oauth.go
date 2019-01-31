package client

import (
	"context"
	oauthv1 "github.com/openshift/api/oauth/v1"
	"k8s.io/apimachinery/pkg/types"
)

// CreateOauthClient creates the OauthClient.
func (c *clientImpl) CreateOAuthClient(oc *oauthv1.OAuthClient) error {
	return c.Client.Create(context.Background(), oc)
}

// GetOauthClient returns the existing OAuthClient.
func (c *clientImpl) GetOAuthClient(name string) (*oauthv1.OAuthClient, error) {
	oc := &oauthv1.OAuthClient{}
	if err := c.Client.Get(context.Background(), types.NamespacedName{Name: name}, oc); err != nil {
		return nil, err
	}
	return oc, nil
}
