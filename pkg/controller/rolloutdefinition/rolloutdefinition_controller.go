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

	appsv1alpha1 "github.com/openkruise/kruise/pkg/apis/apps/v1alpha1"
	"github.com/openkruise/kruise/pkg/dynamic"
	dynamicclientset "github.com/openkruise/kruise/pkg/dynamic/clientset"
	dynamicdiscovery "github.com/openkruise/kruise/pkg/dynamic/discovery"
	dynamicinformer "github.com/openkruise/kruise/pkg/dynamic/informer"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
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

	dynamic, err := dynamic.NewDynamic()
	if err != nil {
		return nil, err
	}

	return &ReconcileRolloutDefinition{
		Client:       mgr.GetClient(),
		scheme:       mgr.GetScheme(),
		PathTable:    make(map[appsv1alpha1.ControlResource]*appsv1alpha1.Path),
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
	PathTable    map[appsv1alpha1.ControlResource]*appsv1alpha1.Path
	resources    *dynamicdiscovery.ResourceMap
	dynClient    *dynamicclientset.Clientset
	dynInformers *dynamicinformer.SharedInformerFactory
}

// Reconcile reads that state of the cluster for a RolloutDefinition object and makes changes based on the state read
// and what is in the RolloutDefinition.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  The scaffolding writes
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
	klog.Infof("qwkLog：Action rolloutDefinition reconcile %v", rolloutDef.Spec.ControlResource)

	contrlResource := rolloutDef.Spec.ControlResource
	if path, ok := r.PathTable[contrlResource]; ok {
		// The path was already stored.
		if apiequality.Semantic.DeepEqual(rolloutDef.Spec.Path, path) {
			// Nothing has changed.
			klog.Infof("qwkLog：nothing to do for %v", rolloutDef.Spec.ControlResource)
			return reconcile.Result{}, nil
		}
	} else {
		// TODO: create a new controller

		klog.Infof("qwkLog：begin create a new controller for %v", rolloutDef.Spec.ControlResource)
		/*resourceClient, err := r.dynClient.Resource(rolloutDef.Spec.ControlResource.APIVersion, rolloutDef.Spec.ControlResource.Resource)
		if err != nil {
			return reconcile.Result{}, err
		}*/
		/*resourceInformer, err := r.dynInformers.Resource(rolloutDef.Spec.ControlResource.APIVersion, rolloutDef.Spec.ControlResource.Resource)
		if err != nil {
			return reconcile.Result{}, err
		}
		klog.Infof("qwkLog：namespace: %v + name: %v", request.Namespace, request.Name)
		resource, err := resourceInformer.Lister().Get(request.Namespace, request.Name)
		klog.Infof("qwkLog：dynamic resource: %v", resource)
		var val interface{} = resource.Object
		if m, ok := val.(map[string]interface{}); ok {
			val, ok = m["spec"]
			if ok {
				klog.Infof("qwkLog：spec val: %v", val)
			}
		}
		klog.Infof("qwkLog：dynamic resource spec object: %v", resource.Object["spec"])

		klog.Infof("qwkLog：end create a new controller for %v", rolloutDef.Spec.ControlResource)*/

	}
	// update pathTable
	r.PathTable[contrlResource] = &rolloutDef.Spec.Path
	klog.Infof("qwkLog：update map:  %v : %v", contrlResource.Resource, r.PathTable[contrlResource].SpecPath.MaxUnavailable)

	return reconcile.Result{}, nil
}
