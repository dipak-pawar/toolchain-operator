package toolchainenabler

import (
	"context"
	"net/url"
	"time"

	"fmt"

	codereadyv1alpha1 "github.com/fabric8-services/toolchain-operator/pkg/apis/codeready/v1alpha1"
	"github.com/fabric8-services/toolchain-operator/pkg/client"
	oauthv1 "github.com/openshift/api/oauth/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"

	clusterclient "github.com/fabric8-services/fabric8-cluster-client/cluster"
	"github.com/fabric8-services/fabric8-common/httpsupport"
	"github.com/fabric8-services/toolchain-operator/pkg/cluster"
	"github.com/fabric8-services/toolchain-operator/pkg/config"
	"github.com/fabric8-services/toolchain-operator/pkg/online_registration"
	"github.com/fabric8-services/toolchain-operator/pkg/secret"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	errs "github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_toolchainenabler")

const (
	TCSecretName      = "toolChainSecretName"
	TCClientID        = "tc.client.id"
	TCClientSecret    = "tc.client.secret"
	SelfProvisioner   = "system:toolchain-sre:self-provisioner"
	DsaasClusterAdmin = "system:toolchain-sre:dsaas-cluster-admin"
)

// Add creates a new ToolChainEnabler Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager, infraCache cache.Cache) error {

	configuration, err := config.NewConfiguration()
	if err != nil {
		return errs.Wrapf(err, "something went wrong while creating configuration")
	}

	reconciler := &ReconcileToolChainEnabler{client: client.NewClient(mgr.GetClient()), scheme: mgr.GetScheme(), config: configuration, cache: infraCache}

	// Create a new controller
	c, err := controller.New("toolchainenabler-controller", mgr, controller.Options{Reconciler: reconciler})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource ToolChainEnabler
	if err := c.Watch(&source.Kind{Type: &codereadyv1alpha1.ToolChainEnabler{}}, &handler.EnqueueRequestForObject{}); err != nil {
		return err
	}

	// Watch for changes to secondary resource Service Account and requeue the owner ToolChainEnabler
	enqueueRequestForOwner := &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &codereadyv1alpha1.ToolChainEnabler{},
	}

	if err := c.Watch(&source.Kind{Type: &corev1.ServiceAccount{}}, enqueueRequestForOwner); err != nil {
		return err
	}

	if err := c.Watch(&source.Kind{Type: &rbacv1.ClusterRoleBinding{}}, enqueueRequestForOwner); err != nil {
		return err
	}

	if err := c.Watch(&source.Kind{Type: &oauthv1.OAuthClient{}}, enqueueRequestForOwner); err != nil {
		return err
	}

	o := &corev1.ServiceAccount{}
	obj := o.DeepCopyObject()
	informer, err := infraCache.GetInformer(obj)
	if err != nil {
		return fmt.Errorf("failed to get informer for %v: %v", obj, err)
	}
	if err := c.Watch(&source.Informer{Informer: informer}, handler.Funcs{
		CreateFunc:  func(e event.CreateEvent, q workqueue.RateLimitingInterface) { q.Add(reconcileRequest(e.Meta)) },
		UpdateFunc:  func(e event.UpdateEvent, q workqueue.RateLimitingInterface) { q.Add(reconcileRequest(e.MetaNew)) },
		DeleteFunc:  func(e event.DeleteEvent, q workqueue.RateLimitingInterface) { q.Add(reconcileRequest(e.Meta)) },
		GenericFunc: func(e event.GenericEvent, q workqueue.RateLimitingInterface) { q.Add(reconcileRequest(e.Meta)) },
	}, predicate.Funcs{
		CreateFunc:  func(e event.CreateEvent) bool { return isOpenshiftInfraServiceAccount(e.Meta.GetName()) },
		DeleteFunc:  func(e event.DeleteEvent) bool { return isOpenshiftInfraServiceAccount(e.Meta.GetName()) },
		UpdateFunc:  func(e event.UpdateEvent) bool { return isOpenshiftInfraServiceAccount(e.MetaNew.GetName()) },
		GenericFunc: func(e event.GenericEvent) bool { return isOpenshiftInfraServiceAccount(e.Meta.GetName()) },
	}); err != nil {
		return fmt.Errorf("failed to create watch for %v: %v", obj, err)
	}

	return nil
}

func isOpenshiftInfraServiceAccount(name string) bool {
	return name == online_registration.ServiceAccountName
}

func reconcileRequest(objMeta metav1.Object) reconcile.Request {
	return reconcile.Request{NamespacedName: types.NamespacedName{
		Name:      objMeta.GetName(),
		Namespace: objMeta.GetNamespace(),
	}}
}

var _ reconcile.Reconciler = &ReconcileToolChainEnabler{}

// ReconcileToolChainEnabler reconciles a ToolChainEnabler object
type ReconcileToolChainEnabler struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
	config *config.Configuration

	// maintaining secondary cache for openshift-infra namespace to do necessary actions for Service Account
	cache cache.Cache
}

// Reconcile reads that state of the cluster for a ToolChainEnabler object and makes changes based on the state read
// and what is in the ToolChainEnabler.Spec
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileToolChainEnabler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling ToolChainEnabler")

	// Fetch the ToolChainEnabler instance
	instance := &codereadyv1alpha1.ToolChainEnabler{}
	namespacedName := request.NamespacedName

	// find toolchainenabler custom resource object for operator ns only and overwrite ns  for cluster scoped resources
	// like OAuthClient, ClusterRoleBinding as you can't get namespace from it's request event
	if request.Namespace != online_registration.Namespace {
		if request.Namespace == "" {
			log.Info(`couldn't find namespace in the request, getting it from env variable "WATCH_NAMESPACE"`)
			ns, err := k8sutil.GetWatchNamespace()
			if err != nil {
				log.Error(err, "can't reconcile request coming from cluster scoped resources event")
				return reconcile.Result{}, nil
			}
			namespacedName = types.NamespacedName{Namespace: ns, Name: request.Name}
		}
		if err := r.client.Get(context.TODO(), namespacedName, instance); err != nil {
			if errors.IsNotFound(err) {
				log.Info("Requeueing request doesn't start as couldn't find requested object or stopped as requested object could have been deleted")
				// Request object not found, could have been deleted after reconcile request.
				// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
				// Return and don't requeue
				return reconcile.Result{}, nil
			}
			// Error reading the object - requeue the request.
			return reconcile.Result{}, err
		}
	}

	// create service account online-registration in openshift-infra namespace
	if err := online_registration.EnsureServiceAccount(r.client, r.cache); err != nil {
		return reconcile.Result{}, err
	}

	// create clusterrolebinding online-registration to service account online-registration
	if err := online_registration.EnsureClusterRoleBinding(r.client); err != nil {
		return reconcile.Result{}, err
	}

	if request.Namespace == online_registration.Namespace {
		// do not reconcile as online-registration specific logic is already reconciled, it's been called by 'openshift-infra' ns informer
		return reconcile.Result{}, nil
	}

	// Create SA
	if err := r.ensureSA(instance); err != nil {
		return reconcile.Result{}, err
	}

	if err := r.ensureClusterRoleBinding(instance, config.SAName, instance.Namespace); err != nil {
		return reconcile.Result{}, err
	}

	if err := r.ensureOAuthClient(instance); err != nil {
		return reconcile.Result{}, err
	}

	cfg, err := createConfig(r.client, namespacedName.Namespace, instance.Spec)
	if err != nil {
		return reconcile.Result{}, err
	}

	clusterData, err := r.clusterInfo(namespacedName.Namespace, cfg)
	if err != nil {
		return reconcile.Result{}, err
	}

	if err := r.saveClusterConfiguration(clusterData, cfg); err != nil {
		log.Error(err, "failed to save cluster configuration in cluster service", "cluster_service_url", cfg.GetClusterServiceURL())
		// requeue after 5 seconds if failed while calling remote cluster service
		return reconcile.Result{RequeueAfter: 5 * time.Second}, nil
	}

	reqLogger.Info("Skipping reconcile as cluster configuration has been updated to cluster management service successfully")
	return reconcile.Result{}, nil
}

// ensureSA creates Service Account if not exists
func (r ReconcileToolChainEnabler) ensureSA(tce *codereadyv1alpha1.ToolChainEnabler) error {
	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      config.SAName,
			Namespace: tce.Namespace,
		},
	}

	// Set ToolChainEnabler instance as the owner and controller
	if err := controllerutil.SetControllerReference(tce, sa, r.scheme); err != nil {
		return err
	}

	if _, err := r.client.GetServiceAccount(tce.Namespace, config.SAName); err != nil {
		if errors.IsNotFound(err) {
			log.Info("creating a new service account ", "namespace", sa.Namespace, "name", sa.Name)
			if err := r.client.CreateServiceAccount(sa); err != nil {
				return err
			}

			log.Info(fmt.Sprintf("service account %s created successfully", config.SAName))
			return nil
		}
		return errs.Wrapf(err, "failed to get service account %s", config.SAName)
	}
	log.Info(fmt.Sprintf("service account %s already exists", config.SAName))

	return nil
}

// ensureClusterRoleBinding ensures ClusterRoleBinding for Service Account with required roles
func (r *ReconcileToolChainEnabler) ensureClusterRoleBinding(tce *codereadyv1alpha1.ToolChainEnabler, saName, namespace string) error {
	if err := r.bindSelfProvisionerRole(tce, saName, namespace); err != nil {
		return err
	}

	return r.bindDsaasClusterAdminRole(tce, saName, namespace)
}

// bindSelfProvisionerRole creates ClusterRoleBinding for Service Account with self-provisioner cluster role
func (r *ReconcileToolChainEnabler) bindSelfProvisionerRole(tce *codereadyv1alpha1.ToolChainEnabler, saName, namespace string) error {
	crb := &rbacv1.ClusterRoleBinding{
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				APIGroup:  "",
				Name:      saName,
				Namespace: namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "self-provisioner",
		},
	}

	crb.SetName(SelfProvisioner)

	// Set ToolChainEnabler instance as the owner and controller
	if err := controllerutil.SetControllerReference(tce, crb, r.scheme); err != nil {
		return err
	}
	if _, err := r.client.GetClusterRoleBinding(SelfProvisioner); err != nil {
		if errors.IsNotFound(err) {
			log.Info(`adding "self-provisioner" cluster role to `, "Service Account", saName)
			if err := r.client.CreateClusterRoleBinding(crb); err != nil {
				return err
			}

			log.Info(fmt.Sprintf("clusterrolebinding %s created successfully", SelfProvisioner))
			return nil
		}
		return errs.Wrapf(err, "failed to get clusterrolebinding %s", SelfProvisioner)
	}

	log.Info(fmt.Sprintf("clusterrolebinding %s already exists", SelfProvisioner))

	return nil
}

// bindDsaasClusterAdminRole creates ClusterRoleBinding for Service Account with dsaas-cluster-admin cluster role
func (r *ReconcileToolChainEnabler) bindDsaasClusterAdminRole(tce *codereadyv1alpha1.ToolChainEnabler, saName, namespace string) error {
	// currently we have defined ClusterRole dsaas-cluster-admin which needs to be create before running this operator.
	// TODO: we should verify this cluster role existence and create if missing.
	crb := &rbacv1.ClusterRoleBinding{
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				APIGroup:  "",
				Name:      saName,
				Namespace: namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "dsaas-cluster-admin",
		},
	}

	crb.SetName(DsaasClusterAdmin)

	// Set ToolChainEnabler instance as the owner and controller
	if err := controllerutil.SetControllerReference(tce, crb, r.scheme); err != nil {
		return err
	}
	if _, err := r.client.GetClusterRoleBinding(DsaasClusterAdmin); err != nil {
		if errors.IsNotFound(err) {
			log.Info(`adding "dsaas-cluster-admin" cluster role to `, "Service Account", saName)
			if err := r.client.CreateClusterRoleBinding(crb); err != nil {
				return err
			}

			log.Info(fmt.Sprintf("clusterrolebinding %s created successfully", DsaasClusterAdmin))
			return nil
		}
		return errs.Wrapf(err, "failed to get clusterrolebinding %s", DsaasClusterAdmin)
	}

	log.Info(fmt.Sprintf("clusterrolebinding %s already exists", DsaasClusterAdmin))

	return nil
}

// ensureOAuthClient creates OAuthClient if not exists
func (r ReconcileToolChainEnabler) ensureOAuthClient(tce *codereadyv1alpha1.ToolChainEnabler) error {
	randomString, err := secret.CreateRandomString(256)
	if err != nil {
		return errs.Wrapf(err, "failed to generate random string to be used as secret for oauthclient")
	}
	var ageSeconds int32
	oc := &oauthv1.OAuthClient{
		ObjectMeta: metav1.ObjectMeta{
			Name: config.OAuthClientName,
		},
		Secret:                   randomString,
		GrantMethod:              oauthv1.GrantHandlerAuto,
		RedirectURIs:             []string{"https://auth.openshift.io/"},
		AccessTokenMaxAgeSeconds: &ageSeconds,
	}

	// Set ToolChainEnabler instance as the owner and controller
	if err := controllerutil.SetControllerReference(tce, oc, r.scheme); err != nil {
		return err
	}

	if _, err = r.client.GetOAuthClient(config.OAuthClientName); err != nil {
		if errors.IsNotFound(err) {
			log.Info("creating", "oauthclient", config.OAuthClientName)
			if err := r.client.CreateOAuthClient(oc); err != nil {
				return err
			}

			log.Info(fmt.Sprintf("oauth client %s created successfully", config.OAuthClientName))
			return nil
		}
		return errs.Wrapf(err, "failed to get oauthclient %s", config.OAuthClientName)
	}

	log.Info(fmt.Sprintf("oauth client %s already exists", config.OAuthClientName))

	return nil
}

func (r ReconcileToolChainEnabler) clusterInfo(ns string, cfg toolChainConfig, options ...cluster.SASecretOption) (*clusterclient.CreateClusterData, error) {
	i := cluster.NewConfigInformer(r.client, ns, cfg.GetClusterName())
	return i.Inform(options...)
}

func (r ReconcileToolChainEnabler) saveClusterConfiguration(data *clusterclient.CreateClusterData, cfg toolChainConfig, options ...httpsupport.HTTPClientOption) error {
	service := cluster.NewClusterService(cfg)
	return service.CreateCluster(context.Background(), data, options...)
}

func createConfig(client client.Client, namespaceName string, spec codereadyv1alpha1.ToolChainEnablerSpec) (tcConfig toolChainConfig, err error) {
	if err = validateURL(spec.AuthURL, "auth service"); err != nil {
		return tcConfig, err
	}
	if err = validateURL(spec.ClusterURL, "cluster service"); err != nil {
		return tcConfig, err
	}
	if spec.ToolChainSecretName == "" {
		return tcConfig, errs.New(fmt.Sprintf("'%s' is empty", TCSecretName))
	}
	secret, err := client.GetSecret(namespaceName, spec.ToolChainSecretName)
	if err != nil {
		return tcConfig, errs.Wrapf(err, "failed to get secret '%s'", spec.ToolChainSecretName)
	}
	if len(secret.Data[TCClientID]) <= 0 {
		return tcConfig, errs.New(fmt.Sprintf("'%s' is empty in secret '%s'", TCClientID, spec.ToolChainSecretName))
	}
	if len(secret.Data[TCClientSecret]) <= 0 {
		return tcConfig, errs.New(fmt.Sprintf("'%s' is empty in secret '%s'", TCClientSecret, spec.ToolChainSecretName))
	}

	tcConfig = toolChainConfig{
		AuthURL:      spec.AuthURL,
		ClusterURL:   spec.ClusterURL,
		ClusterName:  spec.ClusterName,
		ClientID:     string(secret.Data[TCClientID]),
		ClientSecret: string(secret.Data[TCClientSecret]),
	}
	return tcConfig, nil
}

func validateURL(serviceURL, serviceName string) error {
	if serviceURL == "" {
		return errs.New(fmt.Sprintf("'%s' url is empty", serviceName))
	} else {
		u, err := url.Parse(serviceURL)
		if err != nil {
			return errs.Wrapf(err, fmt.Sprintf("invalid url for %s: '%s'", serviceName, serviceURL))
		}
		if u.Host == "" {
			return errs.New(fmt.Sprintf("invalid url '%s' (missing scheme or host?) for: %s", serviceURL, serviceName))
		}
	}
	return nil
}

type toolChainConfig struct {
	AuthURL      string
	ClusterURL   string
	ClusterName  string
	ClientID     string
	ClientSecret string
}

func (c toolChainConfig) GetClusterServiceURL() string {
	return c.ClusterURL
}

func (c toolChainConfig) GetAuthServiceURL() string {
	return c.AuthURL
}

func (c toolChainConfig) GetClientID() string {
	return c.ClientID
}

func (c toolChainConfig) GetClientSecret() string {
	return c.ClientSecret
}

func (c toolChainConfig) GetClusterName() string {
	return c.ClusterName
}
