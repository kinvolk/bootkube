package helm

import (
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
)

type ClientGetter struct {
	config clientcmd.ClientConfig
}

func NewClientGetter(c clientcmd.ClientConfig) *ClientGetter {
	return &ClientGetter{
		config: c,
	}
}

func (c *ClientGetter) ToRESTConfig() (*rest.Config, error) {

	return c.config.ClientConfig()
}

func (c *ClientGetter) ToRawKubeConfigLoader() clientcmd.ClientConfig {
	return c.config
}

func (c *ClientGetter) ToDiscoveryClient() (discovery.CachedDiscoveryInterface, error) {
	config, err := c.ToRESTConfig()
	if err != nil {
		return nil, err
	}

	d, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return nil, err
	}

	return memory.NewMemCacheClient(d), nil
}

func (c *ClientGetter) ToRESTMapper() (meta.RESTMapper, error) {
	d, err := c.ToDiscoveryClient()
	if err != nil {
		return nil, err
	}

	mapper := restmapper.NewDeferredDiscoveryRESTMapper(d)
	expander := restmapper.NewShortcutExpander(mapper, d)

	return expander, nil
}
