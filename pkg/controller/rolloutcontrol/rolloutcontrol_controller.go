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
	"fmt"
	"strings"

	appsv1alpha1 "github.com/openkruise/kruise/pkg/apis/apps/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var Controller controller.Controller

// Add creates a new RolloutControl Controller and adds it to the Manager with default RBAC. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileRolloutControl{
		Client: mgr.GetClient(),
		scheme: mgr.GetScheme(),
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("rolloutcontrol-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// 包
	Controller = c

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
	scheme *runtime.Scheme
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

	obj := &unstructured.Unstructured{}
	obj.SetAPIVersion(rolloutCtl.Spec.Resource.APIVersion)
	obj.SetKind(rolloutCtl.Spec.Resource.Kind)
	obj.SetNamespace(rolloutCtl.Spec.Resource.NameSpace)
	obj.SetName(rolloutCtl.Spec.Resource.Name)

	err = r.Get(context.TODO(), types.NamespacedName{
		Namespace: rolloutCtl.Spec.Resource.NameSpace,
		Name:      rolloutCtl.Spec.Resource.Name,
	}, obj)
	if err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	rolloutDef, err := r.getDefFromControl(rolloutCtl)
	if err != nil {
		return reconcile.Result{}, err
	}
	if rolloutDef == nil {
		klog.Info("can not get rollout definition from control")
		return reconcile.Result{}, nil
	}

	return reconcile.Result{}, r.sync(rolloutCtl, rolloutDef, obj)
}

func (r *ReconcileRolloutControl) sync(rolloutCtl *appsv1alpha1.RolloutControl, rolloutDef *appsv1alpha1.RolloutDefinition, obj *unstructured.Unstructured) error {
	resourcePath := rolloutDef.Spec.Path
	if &resourcePath == nil {
		return fmt.Errorf("have no resourcePath")
	}

	if err := r.updateControlWorkload(rolloutCtl, &resourcePath.SpecPath, obj); err != nil {
		return err
	}

	if err := r.syncStatus(rolloutCtl, &resourcePath.StatusPath, obj); err != nil {
		return err
	}

	return nil
}

func (r *ReconcileRolloutControl) syncStatus(rolloutCtl *appsv1alpha1.RolloutControl, rolloutDefStatusPath *appsv1alpha1.StatusPath, obj *unstructured.Unstructured) error {
	if rolloutDefStatusPath == nil {
		return fmt.Errorf("rollout definition status path is nil")
	}

	// set status replicas
	rolloutControlCopy := rolloutCtl.DeepCopy()
	if rolloutDefStatusPath.Replicas != "" {
		statusReplicasPathArr := strings.Split(rolloutDefStatusPath.Replicas, ".")
		replicas, b, err := unstructured.NestedInt64(obj.Object, statusReplicasPathArr...)
		if err != nil {
			return err
		}
		if b == true {
			rolloutControlCopy.Status.Replicas = int32(replicas)
		} else {
			klog.Info("can't get path field of Replicas")
		}
	}

	// set status current replicas
	if rolloutDefStatusPath.CurrentReplicas != "" {
		statusCurrentReplicasPathArr := strings.Split(rolloutDefStatusPath.CurrentReplicas, ".")
		currentReplicas, b, err := unstructured.NestedInt64(obj.Object, statusCurrentReplicasPathArr...)
		if err != nil {
			return err
		}
		if b == true {
			rolloutControlCopy.Status.CurrentReplicas = int32(currentReplicas)
		} else {
			klog.Info("can't get path field of currentReplicas")
		}
	}

	// set status ready replicas
	if rolloutDefStatusPath.ReadyReplicas != "" {
		readyStatusReplicasPathArr := strings.Split(rolloutDefStatusPath.ReadyReplicas, ".")
		readyReplicas, b, err := unstructured.NestedInt64(obj.Object, readyStatusReplicasPathArr...)
		if err != nil {
			return err
		}
		if b == true {
			rolloutControlCopy.Status.ReadyReplicas = int32(readyReplicas)
		} else {
			klog.Info("can't get path field of ready Replicas")
		}
	}

	// set status ready updated replicas
	if rolloutDefStatusPath.UpdatedReplicas != "" {
		updatedStatusReplicasPathArr := strings.Split(rolloutDefStatusPath.UpdatedReplicas, ".")
		updatedReplicas, b, err := unstructured.NestedInt64(obj.Object, updatedStatusReplicasPathArr...)
		if err != nil {
			return err
		}
		if b == true {
			rolloutControlCopy.Status.UpdatedReplicas = int32(updatedReplicas)
		} else {
			klog.Info("can't get path field of ready Replicas")
		}
	}

	// set status observedGeneration
	if rolloutDefStatusPath.ObservedGeneration != "" {
		observedGenerationPathArr := strings.Split(rolloutDefStatusPath.ObservedGeneration, ".")
		observedGeneration, b, err := unstructured.NestedInt64(obj.Object, observedGenerationPathArr...)
		if err != nil {
			return err
		}
		if b == true {
			rolloutControlCopy.Status.ObservedGeneration = observedGeneration
		} else {
			klog.Info("can't get path field of observed Generation")
		}
	}

	return r.Update(context.TODO(), rolloutControlCopy)
}

func (r *ReconcileRolloutControl) updateControlWorkload(rolloutCtl *appsv1alpha1.RolloutControl, RolloutDefSpecPath *appsv1alpha1.SpecPath, obj *unstructured.Unstructured) error {
	if RolloutDefSpecPath == nil {
		return fmt.Errorf("rollout definition spec path is nil")
	}

	// set paused field
	if RolloutDefSpecPath.Paused != "" {
		pausedPathArr := strings.Split(RolloutDefSpecPath.Paused, ".")
		err := unstructured.SetNestedField(obj.Object, rolloutCtl.Spec.RolloutStrategy.Paused, pausedPathArr...)
		if err != nil {
			return err
		}
	}

	// set partition field
	if RolloutDefSpecPath.Partition != "" {
		partitionPathArr := strings.Split(RolloutDefSpecPath.Partition, ".")
		_, b, err := unstructured.NestedFieldNoCopy(obj.Object, partitionPathArr...)
		if err != nil {
			return err
		}
		if b == true {
			err = unstructured.SetNestedField(obj.Object, rolloutCtl.Spec.RolloutStrategy.Partition, partitionPathArr...)
			if err != nil {
				return err
			}
		} else {
			klog.Info("can't get path field of partition")
		}
	}

	// set maxUnavailable field
	if RolloutDefSpecPath.MaxUnavailable != "" {
		maxUnavailablePathArr := strings.Split(RolloutDefSpecPath.MaxUnavailable, ".")
		_, b, err := unstructured.NestedFieldNoCopy(obj.Object, maxUnavailablePathArr...)
		if err != nil {
			return err
		}
		if b == true {
			err = setNestedField(obj.Object, rolloutCtl.Spec.RolloutStrategy.MaxUnavailable, maxUnavailablePathArr...)
			if err != nil {
				return err
			}
		} else {
			klog.Info("can't get path field of maxUnavailable")
		}
	}

	return r.Update(context.TODO(), obj)
}

func (r *ReconcileRolloutControl) getDefFromControl(rolloutControl *appsv1alpha1.RolloutControl) (*appsv1alpha1.RolloutDefinition, error) {
	rolloutDefs := appsv1alpha1.RolloutDefinitionList{}
	if err := r.List(context.TODO(), &client.ListOptions{}, &rolloutDefs); err != nil {
		return nil, err
	}
	if rolloutControl == nil {
		return nil, fmt.Errorf("rollout contol is nil")
	}

	for _, rolloutDef := range rolloutDefs.Items {
		if rolloutDef.Spec.ControlResource.Kind == rolloutControl.Spec.Resource.Kind && rolloutDef.Spec.ControlResource.APIVersion == rolloutControl.Spec.Resource.APIVersion {
			return &rolloutDef, nil
		}
	}

	return nil, fmt.Errorf("there is no rollout definition")
}

// 如果使用 unstructured.SetNestedField 遇到如下错误
// err: Observed a panic: &errors.errorString{s:"cannot deep copy intstr.IntOrString"} (cannot deep copy intstr.IntOrString)
func setNestedField(obj map[string]interface{}, value interface{}, fields ...string) error {
	m := obj

	for i, field := range fields[:len(fields)-1] {
		if val, ok := m[field]; ok {
			if valMap, ok := val.(map[string]interface{}); ok {
				m = valMap
			} else {
				return fmt.Errorf("value cannot be set because %v is not a map[string]interface{}", jsonPath(fields[:i+1]))
			}
		} else {
			newVal := make(map[string]interface{})
			m[field] = newVal
			m = newVal
		}
	}
	m[fields[len(fields)-1]] = value
	return nil
}

func jsonPath(fields []string) string {
	return "." + strings.Join(fields, ".")
}
