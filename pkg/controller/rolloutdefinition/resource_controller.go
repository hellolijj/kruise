/*
Copyright 2019 The Kruise Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package rolloutdefinition

import (
	"fmt"
	"sync"
	"time"

	"github.com/openkruise/kruise/pkg/apis/apps/v1alpha1"
	dynamicclientset "github.com/openkruise/kruise/pkg/dynamic/clientset"
	dynamicinformer "github.com/openkruise/kruise/pkg/dynamic/informer"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"
)

var (
	KeyFunc = cache.DeletionHandlingMetaNamespaceKeyFunc
)

type resourceController struct {
	resource  *v1alpha1.ControlResource
	dynClient *dynamicclientset.Clientset
	client    *dynamicclientset.ResourceClient
	informer  *dynamicinformer.ResourceInformer

	stopCh, doneCh chan struct{}
	queue          workqueue.RateLimitingInterface
}

func newResourceController(resource *v1alpha1.ControlResource,
	dynClient *dynamicclientset.Clientset,
	dynInformers *dynamicinformer.SharedInformerFactory) (rc *resourceController, err error) {
	resourceInformer, err := dynInformers.Resource(resource.APIVersion, resource.Resource)
	if err != nil {
		return nil, err
	}
	resourceClient, err := dynClient.Resource(resource.APIVersion, resource.Resource)
	if err != nil {
		return nil, err
	}

	rc = &resourceController{
		resource:  resource,
		dynClient: dynClient,
		client:    resourceClient,
		informer:  resourceInformer,
		queue:     workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), resource.APIVersion+resource.Resource),
	}

	return rc, nil
}

func (rc *resourceController) Start() {
	rc.stopCh = make(chan struct{})
	rc.doneCh = make(chan struct{})

	// Install event handlers. resourceControllers can be created at any time,
	// so we have to assume the shared informers are already running. We can't
	// add event handlers in newResourceController() since rc might be incomplete.
	resourceHandlers := cache.ResourceEventHandlerFuncs{
		AddFunc:    rc.enqueueResourceObject,
		UpdateFunc: rc.updateResourceObject,
		DeleteFunc: rc.enqueueResourceObject,
	}
	rc.informer.Informer().AddEventHandler(resourceHandlers)

	go func() {
		defer close(rc.doneCh)
		defer utilruntime.HandleCrash()
		klog.Infof("Starting %v/%v Controller", rc.resource.APIVersion, rc.resource.Resource)
		defer klog.Infof("Starting %v/%v Controller", rc.resource.APIVersion, rc.resource.Resource)
		// Wait for dynamic client and all informers.
		syncFuncs := make([]cache.InformerSynced, 0, 2)
		syncFuncs = append(syncFuncs, rc.dynClient.HasSynced, rc.informer.Informer().HasSynced)
		if !cache.WaitForCacheSync(rc.stopCh, syncFuncs...) {
			utilruntime.HandleError(fmt.Errorf("unable to sync caches for %v/%v controller", rc.resource.APIVersion, rc.resource.Resource))
			klog.Warningf("%v/%v CompositeController cache sync never finished", rc.resource.APIVersion, rc.resource.Resource)
			return
		}
		// 5 workers ought to be enough for anyone.
		var wg sync.WaitGroup
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				wait.Until(rc.worker, time.Second, rc.stopCh)
			}()
		}
		wg.Wait()
	}()
}
func (rc *resourceController) worker() {
	for rc.processNextWorkItem() {
	}
}

func (rc *resourceController) processNextWorkItem() bool {
	key, quit := rc.queue.Get()
	if quit {
		return false
	}
	defer rc.queue.Done(key)

	err := rc.sync(key.(string))
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("failed to sync %v %q: %v", rc.resource.Resource, key, err))
		rc.queue.AddRateLimited(key)
		return true
	}

	rc.queue.Forget(key)
	return true
}

func (rc *resourceController) sync(key string) error {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}

	klog.V(4).Infof("sync %v/%v %v/%v", rc.resource.APIVersion, rc.resource.Resource, namespace, name)
	instance, err := rc.informer.Lister().Get(namespace, name)
	if err != nil {
		return err
	}

	resource := v1alpha1.CompleteResource{
		Kind:       rc.resource.Resource,
		APIVersion: rc.resource.APIVersion,
		NameSpace:  namespace,
		Name:       name,
	}

	// update status of related rolloutCrls
	err = UpdateStatusFromResource(resource, instance.Object)
	if err != nil {
		return err
	}
	return nil
}

func (rc *resourceController) enqueueResourceObject(obj interface{}) {
	key, err := KeyFunc(obj)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("couldn't get key for object %+v: %v", obj, err))
		return
	}
	rc.queue.Add(key)
}

func (rc *resourceController) updateResourceObject(old, cur interface{}) {
	// We used to ignore our own status updates, but we don't anymore.
	// It's sometimes necessary for a hook to see its own status updates
	// so they know that the status was committed to storage.
	// This could cause endless sync hot-loops if your hook always returns a
	// different status (e.g. you have some incrementing counter).
	// Doing that is an anti-pattern anyway because status generation should be
	// idempotent if nothing meaningful has actually changed in the system.
	rc.enqueueResourceObject(cur)
}
