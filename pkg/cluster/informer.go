package cluster

import (
	clusterclient "github.com/fabric8-services/fabric8-cluster-client/cluster"
	"github.com/fabric8-services/toolchain-operator/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var log = logf.Log.WithName("cluster_config_informer")

type informer struct {
	oc          client.Client
	ns          string
	clusterName string
}

type Informer interface {
	Inform(options ...SASecretOption) (*clusterclient.CreateClusterData, error)
}

func NewInformer(oc client.Client, ns string, clusterName string) Informer {
	return informer{oc, ns, clusterName}
}

func (i informer) Inform(options ...SASecretOption) (*clusterclient.CreateClusterData, error) {
	return buildClusterConfiguration(
		name(i),
		appDNS(i),
		apiURL(i),
		oauthClient(i),
		serviceAccount(i, options...),
		tokenProvider(),
		typeOSD(),
	)
}

func buildClusterConfiguration(opts ...configOption) (*clusterclient.CreateClusterData, error) {
	var cluster clusterclient.CreateClusterData
	for _, opt := range opts {
		err := opt(&cluster)
		if err != nil {
			return nil, err
		}
	}
	return &cluster, nil
}
