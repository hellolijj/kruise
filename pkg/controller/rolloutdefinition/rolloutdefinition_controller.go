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
	"context"

	"k8s.io/klog"

	appsv1alpha1 "github.com/hellolijj/kruise/pkg/apis/apps/v1alpha1"
	"github.com/hellolijj/kruise/pkg/dynamic"
	dynamicclientset "github.com/hellolijj/kruise/pkg/dynamic/clientset"
	dynamicdiscovery "github.com/hellolijj/kruise/pkg/dynamic/discovery"
	dynamicinformer "github.com/hellolijj/kruise/pkg/dynamic/informer"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// Add creates a new RolloutDefinition Controller and adds it to the Manager with default RBAC. The Manager will set fields on the Controller
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

	dynamic, err := dynamic.GetDynamic()
	if err != nil {
		return nil, err
	}

	return &ReconcileRolloutDefinition{
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
	c, err := controller.New("rolloutdefinition-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to RolloutDefinition
	err = c.Watch(&source.Kind{Type: &appsv1alpha1.RolloutDefinition{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileRolloutDefinition{}

// ReconcileRolloutDefinition reconciles a RolloutDefinition object
type ReconcileRolloutDefinition struct {
	client.Client
	scheme       *runtime.Scheme
	resources    *dynamicdiscovery.ResourceMap
	dynClient    *dynamicclientset.Clientset
	dynInformers *dynamicinformer.SharedInformerFactory
}

// Reconcile reads that state of the cluster for a RolloutDefinition object and makes changes based on the state read
// and what is in the RolloutDefinition.Spec
// a Deployment as an example
// +kubebuilder:rbac:groups=apps.kruise.io,resources=rolloutdefinitions,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps.kruise.io,resources=rolloutdefinitions/status,verbs=get;update;patch
func (r *ReconcileRolloutDefinition) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	// Fetch the RolloutDefinition instance
	rolloutDef := &appsv1alpha1.RolloutDefinition{}
	err := r.Get(context.TODO(), request.NamespacedName, rolloutDef)
	if err != nil {
		if errors.IsNotFound(err) {
			// Object not found, return.  Created objects are automatically garbage collected.
			// For additional cleanup logic use finalizers.
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	controlResource := rolloutDef.Spec.ControlResource
	if ResourcePathTable.Get(controlResource.APIVersion, controlResource.Resource) == nil {
		klog.Infof("create a new resource controller for %v/%v", controlResource.APIVersion, controlResource.Resource)
		rc, err := newResourceController(&controlResource, r.dynClient, r.dynInformers)
		if err != nil {
			return reconcile.Result{}, err
		}
		rc.Start()
	}

	// update pathTable
	ResourcePathTable.Set(controlResource, &rolloutDef.Spec.Path)

	return reconcile.Result{}, nil
}
