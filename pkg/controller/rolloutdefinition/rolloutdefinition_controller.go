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
	appsv1alpha1 "github.com/openkruise/kruise/pkg/apis/apps/v1alpha1"
	"github.com/openkruise/kruise/pkg/controller/rolloutcontrol"
	"github.com/openkruise/kruise/pkg/util/gate"
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
	if !gate.ResourceEnabled(&appsv1alpha1.RolloutDefinition{}) {
		return nil
	}
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileRolloutDefinition{
		Client: mgr.GetClient(),
		scheme: mgr.GetScheme(),
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

	return nil
}

var _ reconcile.Reconciler = &ReconcileRolloutDefinition{}

// ReconcileRolloutDefinition reconciles a RolloutDefinition object
type ReconcileRolloutDefinition struct {
	client.Client
	scheme *runtime.Scheme
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
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	err = rolloutcontrol.DoWatch(&rolloutDef.Spec.ControlResource, r.Client)
	if err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, err
}
