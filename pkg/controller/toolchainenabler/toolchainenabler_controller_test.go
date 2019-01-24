package toolchainenabler

import (
	codereadyv1alpha1 "github.com/fabric8-services/toolchain-operator/pkg/apis/codeready/v1alpha1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"testing"

	"context"
	"github.com/fabric8-services/toolchain-operator/pkg/client"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"github.com/stretchr/testify/require"
)

// TestToolChainEnablerController runs ReconcileToolChainEnabler.Reconcile() against a
// fake client that tracks a ToolChainEnabler object.
func TestToolChainEnablerController(t *testing.T) {
	// Set the logger to development mode for verbose logs.
	logf.SetLogger(logf.ZapLogger(true))
	name := "toolchain-enabler"

	// A ToolChainEnabler resource with metadata and spec.
	tce := &codereadyv1alpha1.ToolChainEnabler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
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
		t.Run("With ToolChainEnabler Custom Resource", func(t *testing.T) {
			//given
			// Create a fake client to mock API calls.
			cl := client.NewClient(fake.NewFakeClient(objs...))

			// Create a ReconcileToolChainEnabler object with the scheme and fake client.
			r := &ReconcileToolChainEnabler{client: cl, scheme: s}

			req := reconcileRequest(name)

			//when
			res, err := r.Reconcile(req)

			//then
			require.NoError(t, err, "reconcile is failing")
			assert.False(t, res.Requeue, "reconcile requested requeue request")

			assertSA(t, cl)
			assertClusterRoleBinding(t, cl)
		})

		t.Run("without ToolChainEnabler Custom Resource", func(t *testing.T) {
			//given
			// Create a fake client to mock API calls without any runtime object
			cl := client.NewClient(fake.NewFakeClient())

			// Create a ReconcileToolChainEnabler object with the scheme and fake client.
			r := &ReconcileToolChainEnabler{client: cl, scheme: s}

			req := reconcileRequest(name)

			//when
			res, err := r.Reconcile(req)

			//then
			require.NoError(t, err, "reconcile is failing")
			assert.False(t, res.Requeue, "reconcile requested requeue request")

			sa, err := cl.GetServiceAccount(namespace, saName)
			assert.Error(t, err, "found sa %s", saName)
			assert.Nil(t, sa)

			actual, err := cl.GetClusterRoleBinding(crbName)
			assert.Error(t, err, "found ClusterRoleBinding %s", crbName)
			assert.Nil(t, actual)
		})

	})

	t.Run("SA", func(t *testing.T) {

		t.Run("not exists", func(t *testing.T) {
			//given
			// Create a fake client to mock API calls.
			cl := client.NewClient(fake.NewFakeClient(objs...))

			// Create a ReconcileToolChainEnabler object with the scheme and fake client.
			r := &ReconcileToolChainEnabler{client: cl, scheme: s}

			req := reconcileRequest(name)

			instance := &codereadyv1alpha1.ToolChainEnabler{}
			err := r.client.Get(context.TODO(), req.NamespacedName, instance)
			require.NoError(t, err)

			//when
			err = r.ensureSA(instance)
			//then
			require.NoError(t, err, "failed to create SA %s", saName)
			assertSA(t, cl)
		})

		t.Run("exists", func(t *testing.T) {
			//given
			// Create a fake client to mock API calls.
			cl := client.NewClient(fake.NewFakeClient(objs...))

			// Create a ReconcileToolChainEnabler object with the scheme and fake client.
			r := &ReconcileToolChainEnabler{client: cl, scheme: s}

			req := reconcileRequest(name)

			instance := &codereadyv1alpha1.ToolChainEnabler{}
			err := r.client.Get(context.TODO(), req.NamespacedName, instance)
			require.NoError(t, err)

			//create SA first time
			err = r.ensureSA(instance)
			require.NoError(t, err, "failed to create SA %s", saName)
			assertSA(t, cl)

			//when
			err = r.ensureSA(instance)

			//then
			require.NoError(t, err, "failed to create SA %s", saName)
			assertSA(t, cl)

		})

	})

	t.Run("ClusterRoleBinding", func(t *testing.T) {
		t.Run("not exists", func(t *testing.T) {
			//given
			// Create a fake client to mock API calls.
			cl := client.NewClient(fake.NewFakeClient(objs...))

			// Create a ReconcileToolChainEnabler object with the scheme and fake client.
			r := &ReconcileToolChainEnabler{client: cl, scheme: s}

			req := reconcileRequest(name)

			instance := &codereadyv1alpha1.ToolChainEnabler{}
			err := r.client.Get(context.TODO(), req.NamespacedName, instance)
			require.NoError(t, err)

			//when
			err = r.ensureClusterRoleBinding(instance, saName, namespace)
			//then
			require.NoError(t, err, "failed to create SA %s", saName)
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
			req := reconcileRequest(name)

			instance := &codereadyv1alpha1.ToolChainEnabler{}
			err := r.client.Get(context.TODO(), req.NamespacedName, instance)
			require.NoError(t, err)

			// create ClusterRolebinding first time
			err = r.ensureClusterRoleBinding(instance, saName, namespace)

			require.NoError(t, err, "failed to create SA %s", saName)
			assertClusterRoleBinding(t, cl)

			// when
			err = r.ensureClusterRoleBinding(instance, saName, namespace)

			require.NoError(t, err, "failed to create SA %s", saName)
			assertClusterRoleBinding(t, cl)
		})
	})
}

func assertSA(t *testing.T, cl client.ClientInterface) {
	// Check if Service Account has been created
	sa, err := cl.GetServiceAccount(namespace, saName)
	assert.NoError(t, err, "couldn't find created sa %s in namespace %s", saName, namespace)
	assert.NotNil(t, sa)
}

func assertClusterRoleBinding(t *testing.T, cl client.ClientInterface) {
	// Check Service Account has self-provision ClusterRole
	actual, err := cl.GetClusterRoleBinding(crbName)
	assert.NoError(t, err, "couldn't find ClusterRoleBinding %s", crbName)
	assert.NotNil(t, actual)

	subs := []rbacv1.Subject{
		{
			Kind:      "ServiceAccount",
			APIGroup:  "",
			Name:      saName,
			Namespace: namespace,
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

func reconcileRequest(name string) reconcile.Request {
	return reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      name,
			Namespace: namespace,
		},
	}
}
