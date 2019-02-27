package toolchainenabler

import (
	codereadyv1alpha1 "github.com/fabric8-services/toolchain-operator/pkg/apis/codeready/v1alpha1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"testing"

	"context"
	"fmt"
	"github.com/fabric8-services/toolchain-operator/pkg/client"
	. "github.com/fabric8-services/toolchain-operator/pkg/config"
	. "github.com/fabric8-services/toolchain-operator/test"
	oauthv1 "github.com/openshift/api/oauth/v1"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

const (
	Namespace = "codeready-toolchain"
)

// TestToolChainEnablerController runs ReconcileToolChainEnabler.Reconcile() against a
// fake client that tracks a ToolChainEnabler object.
func TestToolChainEnablerController(t *testing.T) {
	// Set the logger to development mode for verbose logs.
	logf.SetLogger(logf.ZapLogger(true))

	// A ToolChainEnabler resource with metadata and spec.
	tce := &codereadyv1alpha1.ToolChainEnabler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      Name,
			Namespace: Namespace,
		},
		Spec: codereadyv1alpha1.ToolChainEnablerSpec{},
	}
	// Objects to track in the fake client.
	objs := []runtime.Object{
		tce,
	}

	// Register operator types with the runtime scheme.
	s := scheme.Scheme
	s.AddKnownTypes(codereadyv1alpha1.SchemeGroupVersion, tce)

	t.Run("Reconcile", func(t *testing.T) {
		t.Run("without registering openshift specific resources", func(t *testing.T) {
			//given
			// Create a fake client to mock API calls.
			cl := client.NewClient(fake.NewFakeClient(objs...))

			// Create a ReconcileToolChainEnabler object with the scheme and fake client.
			r := &ReconcileToolChainEnabler{client: cl, scheme: s}

			req := reconcileRequest(Name)

			//when
			_, err := r.Reconcile(req)

			//then
			_, oautherr := cl.GetOAuthClient(OAuthClientName)
			assert.EqualError(t, err, fmt.Sprintf("failed to get oauthclient %s: %s", OAuthClientName, oautherr))
		})

		t.Run("with ToolChainEnabler custom resource", func(t *testing.T) {
			// register openshift resource OAuthClient specific schema
			err := oauthv1.Install(s)
			require.NoError(t, err)

			reset := SetEnv(Env("CLUSTER_NAME", "dsaas-stage"), Env("TC_CLIENT_ID", "toolchain"), Env("TC_CLIENT_SECRET", "secret"), Env("AUTH_URL", "http://auth"), Env("CLUSTER_URL", "http://cluster"))
			defer reset()

			//given
			// Create a fake client to mock API calls.
			cl := client.NewClient(fake.NewFakeClient(objs...))

			// Create a ReconcileToolChainEnabler object with the scheme and fake client.
			conf, err := NewConfiguration()
			require.NoError(t, err)

			r := &ReconcileToolChainEnabler{client: cl, scheme: s, config: conf}

			req := reconcileRequest(Name)

			//when
			res, err := r.Reconcile(req)

			//then
			require.NoError(t, err, "reconcile is failing")
			assert.False(t, res.Requeue, "reconcile requested requeue request")

			assertSA(t, cl)
			assertClusterRoleBinding(t, cl)
			assertOAuthClient(t, cl)
		})

		t.Run("without ToolChainEnabler custom resource", func(t *testing.T) {
			//given
			// Create a fake client to mock API calls without any runtime object
			cl := client.NewClient(fake.NewFakeClient())

			// Create a ReconcileToolChainEnabler object with the scheme and fake client.
			r := &ReconcileToolChainEnabler{client: cl, scheme: s}

			req := reconcileRequest(Name)

			//when
			res, err := r.Reconcile(req)

			//then
			require.NoError(t, err, "reconcile is failing")
			assert.False(t, res.Requeue, "reconcile requested requeue request")

			sa, err := cl.GetServiceAccount(Namespace, SAName)
			assert.Error(t, err, "failed to get not found error")
			assert.Nil(t, sa, "found sa %s", SAName)

			actual, err := cl.GetClusterRoleBinding(CRBName)
			assert.Error(t, err, "failed to get not found error")
			assert.Nil(t, actual, "found ClusterRoleBinding %s", CRBName)
		})

	})

	t.Run("SA", func(t *testing.T) {

		t.Run("not exists", func(t *testing.T) {
			//given
			// Create a fake client to mock API calls.
			cl := client.NewClient(fake.NewFakeClient(objs...))

			// Create a ReconcileToolChainEnabler object with the scheme and fake client.
			r := &ReconcileToolChainEnabler{client: cl, scheme: s}

			req := reconcileRequest(Name)

			instance := &codereadyv1alpha1.ToolChainEnabler{}
			err := r.client.Get(context.TODO(), req.NamespacedName, instance)
			require.NoError(t, err)

			//when
			err = r.ensureSA(instance)
			//then
			require.NoError(t, err, "failed to create SA %s", SAName)
			assertSA(t, cl)
		})

		t.Run("exists", func(t *testing.T) {
			//given
			// Create a fake client to mock API calls.
			cl := client.NewClient(fake.NewFakeClient(objs...))

			// Create a ReconcileToolChainEnabler object with the scheme and fake client.
			r := &ReconcileToolChainEnabler{client: cl, scheme: s}

			req := reconcileRequest(Name)

			instance := &codereadyv1alpha1.ToolChainEnabler{}
			err := r.client.Get(context.TODO(), req.NamespacedName, instance)
			require.NoError(t, err)

			//create SA first time
			err = r.ensureSA(instance)
			require.NoError(t, err, "failed to create SA %s", SAName)
			assertSA(t, cl)

			//when
			err = r.ensureSA(instance)

			//then
			require.NoError(t, err, "failed to ensure SA %s", SAName)
			assertSA(t, cl)

		})

		t.Run("fail", func(t *testing.T) {
			//given
			errMsg := "something went wrong while getting sa"
			m := make(map[string]string)
			m["sa"] = errMsg

			cl := NewDummyClient(client.NewClient(fake.NewFakeClient(objs...)), m)

			// Create a ReconcileToolChainEnabler object with the scheme and fake client.
			r := &ReconcileToolChainEnabler{client: cl, scheme: s}

			req := reconcileRequest(Name)

			instance := &codereadyv1alpha1.ToolChainEnabler{}
			err := r.client.Get(context.TODO(), req.NamespacedName, instance)
			require.NoError(t, err)

			//when
			err = r.ensureSA(instance)

			//then
			assert.EqualError(t, err, fmt.Sprintf("failed to get service account %s: %s", SAName, errMsg))
		})

	})

	t.Run("ClusterRoleBinding", func(t *testing.T) {
		t.Run("not exists", func(t *testing.T) {
			//given
			// Create a fake client to mock API calls.
			cl := client.NewClient(fake.NewFakeClient(objs...))

			// Create a ReconcileToolChainEnabler object with the scheme and fake client.
			r := &ReconcileToolChainEnabler{client: cl, scheme: s}

			req := reconcileRequest(Name)

			instance := &codereadyv1alpha1.ToolChainEnabler{}
			err := r.client.Get(context.TODO(), req.NamespacedName, instance)
			require.NoError(t, err)

			//when
			err = r.ensureClusterRoleBinding(instance, SAName, Namespace)
			//then
			require.NoError(t, err, "failed to create ClusterRoleBinding %s", SAName)
			assertClusterRoleBinding(t, cl)
		})

		t.Run("exists", func(t *testing.T) {
			//given
			// Create a fake client to mock API calls.
			cl := client.NewClient(fake.NewFakeClient(objs...))

			// Create a ReconcileToolChainEnabler object with the scheme and fake client.
			r := &ReconcileToolChainEnabler{client: cl, scheme: s}

			// Mock request to simulate Reconcile() being called on an event for a
			// watched resource .
			req := reconcileRequest(Name)

			instance := &codereadyv1alpha1.ToolChainEnabler{}
			err := r.client.Get(context.TODO(), req.NamespacedName, instance)
			require.NoError(t, err)

			// create ClusterRolebinding first time
			err = r.ensureClusterRoleBinding(instance, SAName, Namespace)

			require.NoError(t, err, "failed to create ClusterRoleBinding %s", CRBName)
			assertClusterRoleBinding(t, cl)

			// when
			err = r.ensureClusterRoleBinding(instance, SAName, Namespace)

			require.NoError(t, err, "failed to ensure ClusterRoleBinding %s", CRBName)
			assertClusterRoleBinding(t, cl)
		})

		t.Run("fail", func(t *testing.T) {
			//given
			errMsg := "something went wrong while getting clusterrolebinding"
			m := make(map[string]string)
			m["crb"] = errMsg

			cl := NewDummyClient(client.NewClient(fake.NewFakeClient(objs...)), m)

			// Create a ReconcileToolChainEnabler object with the scheme and fake client.
			r := &ReconcileToolChainEnabler{client: cl, scheme: s}

			// Mock request to simulate Reconcile() being called on an event for a
			// watched resource .
			req := reconcileRequest(Name)

			instance := &codereadyv1alpha1.ToolChainEnabler{}
			err := r.client.Get(context.TODO(), req.NamespacedName, instance)
			require.NoError(t, err)

			err = r.ensureClusterRoleBinding(instance, SAName, Namespace)

			//then
			assert.EqualError(t, err, fmt.Sprintf("failed to get clusterrolebinding %s: %s", CRBName, errMsg))
		})
	})

	t.Run("OAuthClient", func(t *testing.T) {
		// register openshift resource OAuthClient specific schema
		err := oauthv1.Install(s)
		require.NoError(t, err)

		t.Run("not exists", func(t *testing.T) {
			//given
			// Create a fake client to mock API calls.
			cl := client.NewClient(fake.NewFakeClient(objs...))

			// Create a ReconcileToolChainEnabler object with the scheme and fake client.
			r := &ReconcileToolChainEnabler{client: cl, scheme: s}

			req := reconcileRequest(Name)

			instance := &codereadyv1alpha1.ToolChainEnabler{}
			err := r.client.Get(context.TODO(), req.NamespacedName, instance)
			require.NoError(t, err)

			//when
			err = r.ensureOAuthClient(instance)
			//then
			require.NoError(t, err, "failed to create OAuthClient %s", OAuthClientName)
			assertOAuthClient(t, cl)
		})

		t.Run("exists", func(t *testing.T) {
			//given
			// Create a fake client to mock API calls.
			cl := client.NewClient(fake.NewFakeClient(objs...))

			// Create a ReconcileToolChainEnabler object with the scheme and fake client.
			r := &ReconcileToolChainEnabler{client: cl, scheme: s}

			// Mock request to simulate Reconcile() being called on an event for a
			// watched resource .
			req := reconcileRequest(Name)

			instance := &codereadyv1alpha1.ToolChainEnabler{}
			err := r.client.Get(context.TODO(), req.NamespacedName, instance)
			require.NoError(t, err)

			// create OAuthClient first time
			err = r.ensureOAuthClient(instance)

			require.NoError(t, err, "failed to create OAuthClient %s", OAuthClientName)
			assertOAuthClient(t, cl)

			// when
			err = r.ensureOAuthClient(instance)

			require.NoError(t, err, "failed to ensure OAuthClient %s", OAuthClientName)
			assertOAuthClient(t, cl)
		})

		t.Run("fail", func(t *testing.T) {
			//given
			errMsg := "something went wrong while getting oauthclient"
			m := make(map[string]string)
			m["oc"] = errMsg

			cl := NewDummyClient(client.NewClient(fake.NewFakeClient(objs...)), m)

			// Create a ReconcileToolChainEnabler object with the scheme and fake client.
			r := &ReconcileToolChainEnabler{client: cl, scheme: s}

			req := reconcileRequest(Name)

			instance := &codereadyv1alpha1.ToolChainEnabler{}
			err := r.client.Get(context.TODO(), req.NamespacedName, instance)
			require.NoError(t, err)

			//when
			err = r.ensureOAuthClient(instance)
			//then
			assert.Error(t, err, "failed to get oauthclient %s: %s", OAuthClientName, errMsg)
		})
	})
}

func assertSA(t *testing.T, cl client.Client) {
	// Check if Service Account has been created
	sa, err := cl.GetServiceAccount(Namespace, SAName)
	assert.NoError(t, err, "couldn't find created sa %s in namespace %s", SAName, Namespace)
	assert.NotNil(t, sa)
}

func assertClusterRoleBinding(t *testing.T, cl client.Client) {
	// Check Service Account has self-provision ClusterRole
	actual, err := cl.GetClusterRoleBinding(CRBName)
	assert.NoError(t, err, "couldn't find ClusterRoleBinding %s", CRBName)
	assert.NotNil(t, actual)

	subs := []rbacv1.Subject{
		{
			Kind:      "ServiceAccount",
			APIGroup:  "",
			Name:      SAName,
			Namespace: Namespace,
		},
	}
	roleRef := rbacv1.RoleRef{
		APIGroup: "rbac.authorization.k8s.io",
		Kind:     "ClusterRole",
		Name:     "self-provisioner",
	}

	assert.Equal(t, actual.Subjects, subs)
	assert.Equal(t, actual.RoleRef, roleRef)
}

func assertOAuthClient(t *testing.T, cl client.Client) {
	// Check OAuthClient has been created
	actual, err := cl.GetOAuthClient(OAuthClientName)
	assert.NoError(t, err, "couldn't find OAuthClient %s", OAuthClientName)
	assert.NotNil(t, actual)

	require.NotNil(t, actual.AccessTokenMaxAgeSeconds)
	assert.Equal(t, *actual.AccessTokenMaxAgeSeconds, int32(0))

	assert.NotEmpty(t, actual.Secret)
	assert.Equal(t, actual.GrantMethod, oauthv1.GrantHandlerAuto)
	assert.Equal(t, actual.RedirectURIs, []string{"https://auth.openshift.io/"})
}

func reconcileRequest(name string) reconcile.Request {
	return reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      name,
			Namespace: Namespace,
		},
	}
}

type DummyClient struct {
	client.Client
	resources map[string]string
}

func NewDummyClient(k8sClient client.Client, opts map[string]string) client.Client {
	return &DummyClient{k8sClient, opts}
}

func (d *DummyClient) GetServiceAccount(namespace, name string) (*v1.ServiceAccount, error) {
	if msg, ok := d.resources["sa"]; ok {
		return nil, errors.New(msg)
	}
	return d.Client.GetServiceAccount(namespace, name)
}

func (d *DummyClient) GetClusterRoleBinding(name string) (*rbacv1.ClusterRoleBinding, error) {
	if msg, ok := d.resources["crb"]; ok {
		return nil, errors.New(msg)
	}
	return d.Client.GetClusterRoleBinding(name)
}

func (d *DummyClient) GetOAuthClient(name string) (*oauthv1.OAuthClient, error) {
	if msg, ok := d.resources["oc"]; ok {
		return nil, errors.New(msg)
	}
	return d.Client.GetOAuthClient(name)
}
