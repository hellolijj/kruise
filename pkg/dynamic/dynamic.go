package dynamic

import (
	"time"

	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"

	dynamicdiscovery "github.com/openkruise/kruise/pkg/dynamic/discovery"
	"k8s.io/klog"
)

func Start() {
	var config *rest.Config

	config, err := rest.InClusterConfig()
	if err != nil {
		klog.Fatal(err)
	}
	// Periodically refresh discovery to pick up newly-installed resources.
	dc := discovery.NewDiscoveryClientForConfigOrDie(config)
	resources := dynamicdiscovery.NewResourceMap(dc)
	// We don't care about stopping this cleanly since it has no external effects.
	resources.Start(30 * time.Second)
}
