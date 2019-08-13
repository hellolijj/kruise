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
	appsv1 "k8s.io/api/apps/v1"
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

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new RolloutDefinition Controller and adds it to the Manager with default RBAC. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileRolloutDefinition{
		Client:    mgr.GetClient(),
		scheme:    mgr.GetScheme(),
		PathTable: make(map[appsv1alpha1.ControlResource]*appsv1alpha1.Path),
	}
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

	// TODO(user): Modify this to be the types you create
	// Uncomment watch a Deployment created by RolloutDefinition - change this for objects you create
	err = c.Watch(&source.Kind{Type: &appsv1.Deployment{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &appsv1alpha1.RolloutDefinition{},
	})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileRolloutDefinition{}

// ReconcileRolloutDefinition reconciles a RolloutDefinition object
type ReconcileRolloutDefinition struct {
	client.Client
	scheme    *runtime.Scheme
	PathTable map[appsv1alpha1.ControlResource]*appsv1alpha1.Path
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
	klog.Infof("qwkLog：Action rolloutDefinition reconcile %v", rolloutDef)
	klog.Infof("qwkLog：Action rolloutDefinition reconcile %v", rolloutDef.Spec)
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
		klog.Infof("qwkLog：create a new controller for %v", rolloutDef.Spec.ControlResource)
	}
	// update pathTable
	r.PathTable[contrlResource] = &rolloutDef.Spec.Path
	klog.Infof("qwkLog：update map:  %v : %v", contrlResource.Resource, r.PathTable[contrlResource].SpecPath.MaxUnavailable)

	return reconcile.Result{}, nil
}
