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
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/openkruise/kruise/pkg/controller/rolloutdefinition"

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
	/*
		（OK）1、获得 RolloutControl 实例 rollourCtl，取得 spec 字段
		（OK）2、利用 RolloutControl.spec 获得要操作的资源对象 resource
		3、利用 RolloutControl.spec 去 RollourDefinition 中取得 resource 的 Path 路径
		4、利用 Path 路径操作 resource.spec
		5、更新 resource
	*/
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
	/* resourceClient, err := r.dynClient.Resource(rolloutCtl.Spec.Resource.APIVersion, rolloutCtl.Spec.Resource.Kind)
	if err != nil {
		return reconcile.Result{}, err
	} */
	resourceInformer, err := r.dynInformers.Resource(rolloutCtl.Spec.Resource.APIVersion, rolloutCtl.Spec.Resource.Kind)
	if err != nil {
		return reconcile.Result{}, err
	}

	resource, err := resourceInformer.Lister().Get(rolloutCtl.Spec.Resource.NameSpace, rolloutCtl.Spec.Resource.Name)
	if err != nil {
		klog.Infof("can't get resource : %v", err)
	}
	klog.Infof("qwkLog：get dynamic resource: %v", resource)
	if resource == nil {
		return reconcile.Result{}, nil
	}
	resourcePath := rolloutdefinition.ResourcePathTable.Get(rolloutCtl.Spec.Resource.APIVersion, rolloutCtl.Spec.Resource.Kind)
	if resourcePath == nil {
		klog.Info("qwkLog: have no resourcePath")
		return reconcile.Result{}, nil
	}
	pausedPathArr := strings.Split(resourcePath.SpecPath.Paused, ".")
	pausedV, b, err := unstructured.NestedFieldNoCopy(resource.Object, pausedPathArr...)
	if err != nil {
		return reconcile.Result{}, err
	}
	if b == false {
		klog.Info("can't get path field")
		return reconcile.Result{}, nil
	}
	klog.Infof("qwkLog：get paused value: %v", pausedV)

	klog.Info("qwkLog：begin set paused value")
	err = unstructured.SetNestedField(resource.Object, rolloutCtl.Spec.RolloutStrategy.Paused, pausedPathArr...)
	if err != nil {
		return reconcile.Result{}, err
	}
	klog.Info("qwkLog：end set paused value")
	pausedV, b, err = unstructured.NestedFieldNoCopy(resource.Object, pausedPathArr...)
	if err != nil {
		return reconcile.Result{}, err
	}
	if b == false {
		klog.Info("can't get path field  after set")
		return reconcile.Result{}, nil
	}
	klog.Infof("qwkLog：get paused value after set: %v", pausedV)

	return reconcile.Result{}, nil
}
