package online_registration

import (
	"fmt"
	"github.com/fabric8-services/toolchain-operator/pkg/client"
	. "github.com/fabric8-services/toolchain-operator/pkg/config"
	"github.com/fabric8-services/toolchain-operator/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	rbacv1 "k8s.io/api/rbac/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

func TestResourceCreator(t *testing.T) {

	t.Run("SA", func(t *testing.T) {

		t.Run("not exists", func(t *testing.T) {
			//given
			cl := client.NewClient(fake.NewFakeClient())
			r := &resourceCreator{client: cl}

			//when
			err := r.createServiceAccount()

			//then
			require.NoError(t, err, "failed to create SA %s", ServiceAccountName)
			assertSA(t, cl)
		})

		t.Run("exists", func(t *testing.T) {
			//given
			cl := client.NewClient(fake.NewFakeClient())
			r := &resourceCreator{client: cl}

			//create SA first time
			err := r.createServiceAccount()
			require.NoError(t, err, "failed to create SA %s", ServiceAccountName)
			assertSA(t, cl)

			//when
			err = r.createServiceAccount()

			//then
			require.NoError(t, err, "failed to ensure SA %s", ServiceAccountName)
			assertSA(t, cl)

		})

		t.Run("fail", func(t *testing.T) {
			//given
			errMsg := "something went wrong while getting sa"
			m := make(map[string]string)
			m["sa"] = errMsg

			cl := test.NewDummyClient(client.NewClient(fake.NewFakeClient()), m)
			r := &resourceCreator{client: cl}

			//when
			err := r.createServiceAccount()

			//then
			assert.EqualError(t, err, fmt.Sprintf("failed to get service account %s: %s", ServiceAccountName, errMsg))
		})

	})

	t.Run("ClusterRoleBinding", func(t *testing.T) {
		t.Run("not exists", func(t *testing.T) {
			//given
			cl := client.NewClient(fake.NewFakeClient())
			r := &resourceCreator{client: cl}

			//when
			err := r.createClusterRoleBinding()

			//then
			require.NoError(t, err, "failed to create ClusterRoleBinding %s", ClusterRoleBindingName)
			assertClusterRoleBinding(t, cl)
		})

		t.Run("exists", func(t *testing.T) {
			//given
			cl := client.NewClient(fake.NewFakeClient())
			r := &resourceCreator{client: cl}

			//when
			err := r.createClusterRoleBinding()

			//then
			require.NoError(t, err, "failed to create ClusterRoleBinding %s", ClusterRoleBindingName)
			assertClusterRoleBinding(t, cl)

			// when
			err = r.createClusterRoleBinding()

			require.NoError(t, err, "failed to ensure ClusterRoleBinding %s", ClusterRoleBindingName)
			assertClusterRoleBinding(t, cl)
		})

		t.Run("fail", func(t *testing.T) {
			//given
			errMsg := "something went wrong while getting clusterrolebinding"
			m := make(map[string]string)
			m["crb"] = errMsg

			cl := test.NewDummyClient(client.NewClient(fake.NewFakeClient()), m)
			r := &resourceCreator{client: cl}

			//when
			err := r.createClusterRoleBinding()

			//then
			assert.EqualError(t, err, fmt.Sprintf("failed to get clusterrolebinding %s: %s", ClusterRoleBindingName, errMsg))
		})
	})
}

func TestEnsureResources(t *testing.T) {
	// given
	cl := client.NewClient(fake.NewFakeClient())

	// when
	err := EnsureResources(cl)
	require.NoError(t, err, "failed to ensure SA %s", SAName)

	//then
	assertSA(t, cl)
	assertClusterRoleBinding(t, cl)
}

func assertSA(t *testing.T, cl client.Client) {
	// Check if service account has been created
	sa, err := cl.GetServiceAccount(Namespace, ServiceAccountName)
	assert.NoError(t, err, "couldn't find created sa %s in namespace %s", ServiceAccountName, Namespace)
	assert.NotNil(t, sa)
}

func assertClusterRoleBinding(t *testing.T, cl client.Client) {
	// Check service account has online-registration clusterrole
	actual, err := cl.GetClusterRoleBinding(ClusterRoleBindingName)
	assert.NoError(t, err, "couldn't find ClusterRoleBinding %s", ClusterRoleBindingName)
	assert.NotNil(t, actual)

	subs := []rbacv1.Subject{
		{
			Kind: "ServiceAccount",
			Name: ServiceAccountName,
		},
	}

	roleRef := rbacv1.RoleRef{
		APIGroup: "rbac.authorization.k8s.io",
		Kind:     "ClusterRole",
		Name:     "online-registration",
	}

	assert.Equal(t, actual.Subjects, subs)
	assert.Equal(t, actual.RoleRef, roleRef)
}
