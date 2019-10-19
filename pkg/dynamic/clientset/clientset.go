package clientset

import (
	"fmt"

	dynamicdiscovery "github.com/hellolijj/kruise/pkg/dynamic/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
)

type Clientset struct {
	config    rest.Config
	resources *dynamicdiscovery.ResourceMap
	dc        dynamic.Interface
}

func New(config *rest.Config, resources *dynamicdiscovery.ResourceMap) (*Clientset, error) {
	dc, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("can't create dynamic client when creating clientset: %v", err)
	}
	return &Clientset{
		config:    *config,
		resources: resources,
		dc:        dc,
	}, nil
}

func (cs *Clientset) HasSynced() bool {
	return cs.resources.HasSynced()
}

func (cs *Clientset) Resource(apiVersion, resource string) (*ResourceClient, error) {
	// Look up the requested resource in discovery.
	apiResource := cs.resources.Get(apiVersion, resource)
	if apiResource == nil {
		return nil, fmt.Errorf("discovery: can't find resource %s in apiVersion %s", resource, apiVersion)
	}
	return cs.resource(apiResource), nil
}

func (cs *Clientset) Kind(apiVersion, kind string) (*ResourceClient, error) {
	// Look up the requested resource in discovery.
	apiResource := cs.resources.GetKind(apiVersion, kind)
	if apiResource == nil {
		return nil, fmt.Errorf("discovery: can't find kind %s in apiVersion %s", kind, apiVersion)
	}
	return cs.resource(apiResource), nil
}

func (cs *Clientset) resource(apiResource *dynamicdiscovery.APIResource) *ResourceClient {
	client := cs.dc.Resource(apiResource.GroupVersionResource())
	return &ResourceClient{
		ResourceInterface: client,
		APIResource:       apiResource,
		rootClient:        client,
	}
}

// ResourceClient is a combination of APIResource and a dynamic Client.
//
// Passing this around makes it easier to write code that deals with arbitrary
// resource types and often needs to know the API discovery details.
// This wrapper also adds convenience functions that are useful for any client.
//
// It can be used on either namespaced or cluster-scoped resources.
// Call Namespace() to return a client that's scoped down to a given namespace.
type ResourceClient struct {
	dynamic.ResourceInterface
	*dynamicdiscovery.APIResource

	rootClient dynamic.NamespaceableResourceInterface
}

// Namespace returns a copy of the ResourceClient with the client namespace set.
//
// This can be chained to set the namespace to something else.
// Pass "" to return a client with the namespace cleared.
// If the resource is cluster-scoped, this is a no-op.
func (rc *ResourceClient) Namespace(namespace string) *ResourceClient {
	// Ignore the namespace if the resource is cluster-scoped.
	if !rc.Namespaced {
		return rc
	}
	// Reset to cluster-scoped if provided namespace is empty.
	ri := dynamic.ResourceInterface(rc.rootClient)
	if namespace != "" {
		ri = rc.rootClient.Namespace(namespace)
	}
	return &ResourceClient{
		ResourceInterface: ri,
		APIResource:       rc.APIResource,
		rootClient:        rc.rootClient,
	}
}
