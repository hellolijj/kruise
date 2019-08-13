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

package rolloutcontrol

import (
	"context"

	"k8s.io/klog"

	"github.com/openkruise/kruise/pkg/dynamic"

	appsv1alpha1 "github.com/openkruise/kruise/pkg/apis/apps/v1alpha1"
	dynamicclientset "github.com/openkruise/kruise/pkg/dynamic/clientset"
	dynamicdiscovery "github.com/openkruise/kruise/pkg/dynamic/discovery"
	dynamicinformer "github.com/openkruise/kruise/pkg/dynamic/informer"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// Add creates a new RolloutControl Controller and adds it to the Manager with default RBAC. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	r, err := newReconciler(mgr)
	if err != nil {
		return err
	}
	return add(mgr, r)
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) (reconcile.Reconciler, error) {
	dynamic, err := dynamic.NewDynamic()
	if err != nil {
		return nil, err
	}
	return &ReconcileRolloutControl{
		Client:       mgr.GetClient(),
		scheme:       mgr.GetScheme(),
		resources:    dynamic.Resources,
		dynClient:    dynamic.DynClient,
		dynInformers: dynamic.DynInformers,
	}, nil
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("rolloutcontrol-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to RolloutControl
	err = c.Watch(&source.Kind{Type: &appsv1alpha1.RolloutControl{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileRolloutControl{}

// ReconcileRolloutControl reconciles a RolloutControl object
type ReconcileRolloutControl struct {
	client.Client
	scheme       *runtime.Scheme
	resources    *dynamicdiscovery.ResourceMap
	dynClient    *dynamicclientset.Clientset
	dynInformers *dynamicinformer.SharedInformerFactory
}

// Reconcile reads that state of the cluster for a RolloutControl object and makes changes based on the state read
// and what is in the RolloutControl.Spec
// Automatically generate RBAC rules to allow the Controller to read and write Deployments
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps.kruise.io,resources=rolloutcontrols,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps.kruise.io,resources=rolloutcontrols/status,verbs=get;update;patch
func (r *ReconcileRolloutControl) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	// Fetch the RolloutControl instance
	rolloutCtl := &appsv1alpha1.RolloutControl{}
	err := r.Get(context.TODO(), request.NamespacedName, rolloutCtl)
	if err != nil {
		if errors.IsNotFound(err) {
			// Object not found, return.  Created objects are automatically garbage collected.
			// For additional cleanup logic use finalizers.
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}
	klog.Infof("qwkLog：begin RolloutControl for %v", rolloutCtl.Spec.Resource)
	/*resourceClient, err := r.dynClient.Resource(rolloutCtl.Spec.Resource.APIVersion, rolloutCtl.Spec.Resource.Kind)
	if err != nil {
		return reconcile.Result{}, err
	}*/
	resourceInformer, err := r.dynInformers.Resource(rolloutCtl.Spec.Resource.APIVersion, rolloutCtl.Spec.Resource.Kind)
	if err != nil {
		return reconcile.Result{}, err
	}
	klog.Infof("qwkLog：rolloutCtl:( %v + %v )", rolloutCtl.Spec.Resource.NameSpace, rolloutCtl.Spec.Resource.Name)
	resource, err := resourceInformer.Lister().Get(rolloutCtl.Spec.Resource.NameSpace, rolloutCtl.Spec.Resource.Name)
	klog.Infof("qwkLog：dynamic resource: %v", resource)
	var val interface{} = resource.Object
	if m, ok := val.(map[string]interface{}); ok {
		val, ok = m["spec"]
		if ok {
			klog.Infof("qwkLog：spec val: %v", val)
		}
	}
	klog.Infof("qwkLog：dynamic resource spec object: %v", resource.Object["spec"])

	return reconcile.Result{}, nil
}
