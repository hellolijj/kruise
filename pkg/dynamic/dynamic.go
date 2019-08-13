package dynamic

import (
	"time"

	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"

	dynamicclientset "github.com/openkruise/kruise/pkg/dynamic/clientset"
	dynamicdiscovery "github.com/openkruise/kruise/pkg/dynamic/discovery"
	dynamicinformer "github.com/openkruise/kruise/pkg/dynamic/informer"
)

type Dynamic struct {
	Resources    *dynamicdiscovery.ResourceMap
	DynClient    *dynamicclientset.Clientset
	DynInformers *dynamicinformer.SharedInformerFactory
}

func NewDynamic() (dynamic *Dynamic, err error) {
	informerRelist := 30 * time.Minute
	discoveryInterval := 30 * time.Second
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	// Periodically refresh discovery to pick up newly-installed resources.
	dc := discovery.NewDiscoveryClientForConfigOrDie(config)
	resources := dynamicdiscovery.NewResourceMap(dc)
	resources.Start(discoveryInterval)

	// Create dynamic clientset (factory for dynamic clients).
	dynClient, err := dynamicclientset.New(config, resources)
	if err != nil {
		return nil, err
	}
	// Create dynamic informer factory (for sharing dynamic informers).
	dynInformers := dynamicinformer.NewSharedInformerFactory(dynClient, informerRelist)

	dynamic = &Dynamic{
		Resources:    resources,
		DynClient:    dynClient,
		DynInformers: dynInformers,
	}

	return dynamic, nil
}
