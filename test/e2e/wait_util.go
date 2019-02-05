package e2e

import (
	"github.com/fabric8-services/toolchain-operator/pkg/client"
	"github.com/fabric8-services/toolchain-operator/pkg/controller/toolchainenabler"
	oauthv1 "github.com/openshift/api/oauth/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"reflect"
	"testing"
	"time"
)

const (
	retryInterval        = time.Second * 5
	timeout              = time.Second * 60
	cleanupRetryInterval = time.Second * 1
	cleanupTimeout       = time.Second * 5
)

func waitForServiceAccount(t *testing.T, operatorClient client.Client, namespace string) error {
	return wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		sa, err := operatorClient.GetServiceAccount(namespace, toolchainenabler.SAName)
		if err != nil {
			if apierrors.IsNotFound(err) {
				t.Logf("Waiting for availability of service account %s in namespace %s \n", toolchainenabler.SAName, namespace)
				return false, nil
			}
			return false, err
		}

		if sa != nil {
			t.Logf("Found service account %s in namespace %s \n", toolchainenabler.SAName, namespace)
			return true, nil
		}

		t.Logf("Waiting for service account %s \n", toolchainenabler.SAName)
		return false, nil
	})
}

func waitForClusterRoleBinding(t *testing.T, operatorClient client.Client) error {
	return wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		crb, err := operatorClient.GetClusterRoleBinding(toolchainenabler.CRBName)
		if err != nil {
			if apierrors.IsNotFound(err) {
				t.Logf("Waiting for availability of %s cluster role binding\n", toolchainenabler.CRBName)
				return false, nil
			}
			return false, err
		}

		if crb != nil {
			t.Logf("Found cluster role binding %s \n", toolchainenabler.CRBName)
			return true, nil
		}

		t.Logf("Waiting for cluster role binding %s \n", toolchainenabler.CRBName)
		return false, nil
	})
}

func waitForOauthClient(t *testing.T, operatorClient client.Client) error {
	return wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		oc, err := operatorClient.GetOAuthClient(toolchainenabler.OAuthClientName)
		if err != nil {
			if apierrors.IsNotFound(err) {
				t.Logf("Waiting for availability of oauth client %s \n", toolchainenabler.OAuthClientName)
				return false, nil
			}
			return false, err
		}

		if !reflect.DeepEqual(oauthv1.OAuthClient{}, *oc) {
			t.Logf("Found oauth client %s \n", toolchainenabler.OAuthClientName)
			return true, nil
		}
		t.Logf("Waiting for availability of %s oauth client \n", toolchainenabler.OAuthClientName)
		return false, nil
	})
}

func waitForOauthClientToDelete(t *testing.T, operatorClient client.Client) error {
	return wait.Poll(retryInterval, timeout, func() (bool, error) {
		t.Logf("waiting for oauth client %s to delete", toolchainenabler.OAuthClientName)
		_, err := operatorClient.GetOAuthClient(toolchainenabler.OAuthClientName)
		if apierrors.IsNotFound(err) {
			t.Logf("deleted %s", toolchainenabler.OAuthClientName)
			return true, nil
		}
		if err != nil {
			t.Logf("err trying to delete oauth client %s: %s", toolchainenabler.OAuthClientName, err)
			return false, err
		}
		return false, nil
	})
}

func waitForServiceAccountToDelete(t *testing.T, operatorClient client.Client, namespace string) error {
	return wait.Poll(retryInterval, timeout, func() (bool, error) {
		t.Logf("waiting for service account %s to delete", toolchainenabler.SAName)
		_, err := operatorClient.GetServiceAccount(namespace, toolchainenabler.SAName)
		if apierrors.IsNotFound(err) {
			t.Logf("deleted service account %s/%s", namespace, toolchainenabler.SAName)
			return true, nil
		}
		if err != nil {
			t.Logf("err trying to delete service account %s/%s: %s", namespace, toolchainenabler.SAName, err)
			return false, err
		}
		return false, nil
	})
}

func waitForClusterRoleBindingToDelete(t *testing.T, operatorClient client.Client) error {
	return wait.Poll(retryInterval, timeout, func() (bool, error) {
		t.Logf("waiting for cluster role binding %s to delete", toolchainenabler.SAName)
		_, err := operatorClient.GetClusterRoleBinding(toolchainenabler.CRBName)
		if apierrors.IsNotFound(err) {
			t.Logf("deleted cluster role binding %s", toolchainenabler.CRBName)
			return true, nil
		}
		if err != nil {
			t.Logf("err trying to delete cluster role binding %s: %s", toolchainenabler.CRBName, err)
			return false, err
		}
		return false, nil
	})
}
