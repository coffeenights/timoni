package module

import (
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

type RESTClientGetter struct {
	ClientConfig *rest.Config
	Namespace    string
}

func NewRESTClientGetter() (*RESTClientGetter, error) {
	restConfig, err := config.GetConfig()
	return &RESTClientGetter{ClientConfig: restConfig}, err
}

func (r *RESTClientGetter) ToRESTConfig() (*rest.Config, error) {
	return r.ClientConfig, nil
}

func (r *RESTClientGetter) ToDiscoveryClient() (discovery.CachedDiscoveryInterface, error) {
	c, err := r.ToRESTConfig()
	if err != nil {
		return nil, err
	}
	c.Burst = 100
	discoveryClient, _ := discovery.NewDiscoveryClientForConfig(c)
	return memory.NewMemCacheClient(discoveryClient), nil
}

func (r *RESTClientGetter) ToRESTMapper() (meta.RESTMapper, error) {
	dc, err := r.ToDiscoveryClient()
	if err != nil {
		return nil, err
	}
	return restmapper.NewDeferredDiscoveryRESTMapper(dc), nil
}

func (r *RESTClientGetter) ToRawKubeConfigLoader() clientcmd.ClientConfig {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	loadingRules.DefaultClientConfig = &clientcmd.DefaultClientConfig

	overrides := &clientcmd.ConfigOverrides{ClusterDefaults: clientcmd.ClusterDefaults}
	overrides.Context.Namespace = r.Namespace

	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, overrides)
}
