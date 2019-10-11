package e2e

import (
	"context"
	"github.com/codeready-toolchain/toolchain-operator/pkg/apis/toolchain/v1alpha1"
	"github.com/codeready-toolchain/toolchain-operator/pkg/lib/che"
	"github.com/codeready-toolchain/toolchain-operator/pkg/test/toolchain"
	"github.com/stretchr/testify/require"
	"testing"
	"time"

	"github.com/codeready-toolchain/api/pkg/apis"
	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
	. "github.com/codeready-toolchain/toolchain-operator/pkg/test/k8s"
	. "github.com/codeready-toolchain/toolchain-operator/pkg/test/olm"
	. "github.com/codeready-toolchain/toolchain-operator/pkg/test/toolchain"
)

var (
	retryInterval        = time.Second * 5
	timeout              = time.Second * 60
	cleanupRetryInterval = time.Second * 1
	cleanupTimeout       = time.Second * 5
)

func TestToolchain(t *testing.T) {
	ctx := InitOperator(t)
	defer ctx.Cleanup()

	cheOperatorNs := toolchain.GenerateName("che-op")
	exampleToolChainEnabler := toolchain.NewInstallConfig(cheOperatorNs)
	f := framework.Global
	// use TestCtx's create helper to create the object and add a cleanup function for the new object
	err := f.Client.Create(context.TODO(), exampleToolChainEnabler, &framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	require.NoError(t, err, "failed to create toolchain InstallConfig")


	AssertThatNamespace(t, cheOperatorNs, f)
		Exists().
		HasLabels(che.Labels())

	AssertThatOperatorGroup(t, cheOg.Namespace, cheOg.Name, cl).
		Exists().
		HasSize(1).
		HasSpec(cheOg.Spec)

	AssertThatSubscription(t, cheSub.Name, cl).
		Exists().
		HasSpec(cheSub.Spec)

	AssertThatInstallConfig(t, installConfig.Namespace, installConfig.Name, cl).
		HasConditions(CheSubscriptionCreated("che operator subscription created"))
}

func InitOperator(t *testing.T) *framework.TestCtx {
	icList := &v1alpha1.InstallConfigList{}
	err := framework.AddToFrameworkScheme(apis.AddToScheme, icList)
	require.NoError(t, err, "failed to add custom resource scheme to framework: %v", err)

	t.Parallel()
	ctx := framework.NewTestCtx(t)

	err = ctx.InitializeClusterResources(&framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	require.NoError(t, err, "failed to initialize cluster resources")

	t.Log("Initialized cluster resources")
	namespace, err := ctx.GetNamespace()
	require.NoError(t, err, "failed to get namespace where operator is running")

	// get global framework variables
	f := framework.Global
	// wait for toolchain-operator to be ready
	err = e2eutil.WaitForOperatorDeployment(t, f.KubeClient, namespace, "toolchain-operator", 1, retryInterval, timeout)
	require.NoError(t, err, "failed while waiting for toolchain-operator deployment")

	t.Log("toolchain-operator is ready and running state")

	return ctx
}
