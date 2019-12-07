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

package kanaryanalysis

import (
	"context"
	"fmt"
	appsv1alpha1 "github.com/openkruise/kruise/pkg/apis/apps/v1alpha1"
	"github.com/openkruise/kruise/pkg/controller/kanaryanalysis/utils/status"
	"github.com/openkruise/kruise/pkg/controller/kanaryanalysis/utils/types"
	"github.com/openkruise/kruise/pkg/controller/kanaryanalysis/validation"
	"github.com/openkruise/kruise/pkg/controller/kanaryanalysis/workloadpods"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog"
	"k8s.io/kubernetes/pkg/apis/apps"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"time"
)

var log = logf.Log.WithName("kanaryanalysis-controller")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new KanaryAnalysis Controller and adds it to the Manager with default RBAC. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	newReconciler, err := newReconciler(mgr)
	if err != nil {
		return err
	}
	return add(mgr, newReconciler)
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) (reconcile.Reconciler, error) {
	return &ReconcileKanaryAnalysis{
		Client: mgr.GetClient(),
		scheme: mgr.GetScheme(),
	}, nil
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("kanaryanalysis-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to KanaryAnalysis
	err = c.Watch(&source.Kind{Type: &appsv1alpha1.KanaryAnalysis{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	err = c.Watch(&source.Kind{Type: &appsv1alpha1.StatefulSet{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &appsv1alpha1.UnitedDeployment{},
	})
	if err != nil {
		return err
	}

	err = c.Watch(&source.Kind{Type: &apps.Deployment{}}, &handler.EnqueueRequestForOwner{
		OwnerType:    &appsv1alpha1.KanaryAnalysis{},
		IsController: true,
	})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileKanaryAnalysis{}

// ReconcileKanaryAnalysis reconciles a KanaryAnalysis object
type ReconcileKanaryAnalysis struct {
	client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a KanaryAnalysis object and makes changes based on the state read
// and what is in the KanaryAnalysis.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  The scaffolding writes
// a Deployment as an example
// Automatically generate RBAC rules to allow the Controller to read and write Deployments
// +kubebuilder:rbac:groups=apps.kruise.io,resources=kanaryanalyses,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps.kruise.io,resources=kanaryanalyses/status,verbs=get;update;patch
// 先做获取刚刚更新过的 pod name , 再做金丝雀分析
func (r *ReconcileKanaryAnalysis) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	ka := &appsv1alpha1.KanaryAnalysis{}
	err := r.Get(context.TODO(), request.NamespacedName, ka)
	if err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}
	klog.V(4).Infof("get ka name %s to control %s", ka.Name, ka.Spec.Workload.Name)

	// checkout start time
	if validation.IsValidationStartLine(ka) == false {
		return reconcile.Result{}, fmt.Errorf("wait initial delay")
	}

	// get 正在发布的pod
	workload := workloadpods.New(&ka.Spec)
	if workload == nil {
		return reconcile.Result{}, status.UpdateKanaryAnalysisStatus(r.Client, ka, appsv1alpha1.ErroredKanaryAnalysisConditionType, "workload init error")
	}
	klog.V(4).Infof("get workload: %s", workload.String())
	toCheckPods, err := workload.ListToCheckPods(r.Client, ka)
	if err != nil {
		return nextReconcile(ka, status.UpdateKanaryAnalysisStatus(r.Client, ka, appsv1alpha1.ErroredKanaryAnalysisConditionType, err.Error()))
	}
	if toCheckPods == nil {
		return nextReconcile(ka, status.UpdateKanaryAnalysisStatus(r.Client, ka, appsv1alpha1.ErroredKanaryAnalysisConditionType, "to get update pods nil"))
	}
	klog.V(4).Infof("get workload updated pods count: %d", len(toCheckPods.Pods))

	kaConditionType := getConditionTypeFromCheckPods(ka, toCheckPods)

	if kaConditionType != nil {
		status.UpdateKanaryAnalysisStatus(r.Client, ka, *kaConditionType, "")
	}
	// 已完成部署，不再进行验证
	if *kaConditionType == appsv1alpha1.WorkloadNoUpdatKanaryAnalysisConditionType {
		return nextReconcile(ka, nil)
	}

	// 已完成部署，不再进行验证
	if status.IsWorkloadKanaryAnalysisUpdated(&ka.Status) {
		klog.V(4).Infof("workload %s update finished: %s", ka.Spec.Workload.Name)
		return reconcile.Result{}, nil
	}
	// 验证失败, 不需要再验证
	if status.IsKanaryAnalysisFailed(&ka.Status) {
		workload.SetPausedTrue(r.Client)
		return reconcile.Result{}, fmt.Errorf(status.GetKanaryAnalysisFailedReason(&ka.Status))
	}


	// 验证 pod....
	validationStrategy, err := validation.New(&ka.Spec)
	if err != nil {
		return reconcile.Result{}, status.UpdateKanaryAnalysisStatus(r.Client, ka, appsv1alpha1.ErroredKanaryAnalysisConditionType, err.Error())
	}
	res, err := validationStrategy.Apply(r.Client, ka, toCheckPods.Pods)
	if err != nil {
		return reconcile.Result{}, status.UpdateKanaryAnalysisStatus(r.Client, ka, appsv1alpha1.ErroredKanaryAnalysisConditionType, err.Error())
	}
	klog.V(4).Infof("get validation result: %v", res)

	// 验证成功，持续验证
	if res != nil && res.IsFailed == false {
		return nextReconcile(ka, nil)
	}
	// 验证失败，喊停
	if res != nil && res.IsFailed {
		klog.Info(res.Reason)
		return reconcile.Result{}, status.UpdateKanaryAnalysisStatus(r.Client, ka, appsv1alpha1.FailedKanaryAnalysisConditionType, res.Reason)
	}

	return nextReconcile(ka, nil)
}

func nextReconcile(ka *appsv1alpha1.KanaryAnalysis, err error) (reconcile.Result, error) {
	if ka.Spec.Validation.Interval != nil {
		return reconcile.Result{
			Requeue:      true,
			RequeueAfter: ka.Spec.Validation.ValidationPeriod.Duration,
		}, err
	}
	return reconcile.Result{Requeue:true,RequeueAfter: 4 * time.Second}, err
}

func getConditionTypeFromCheckPods(ka *appsv1alpha1.KanaryAnalysis, t *types.ToCheckPods) *appsv1alpha1.KanaryAnalysisConditionType {
	condition := appsv1alpha1.UnKnownUpdatKanaryAnalysisConditionType
	if t == nil {
		return &condition
	}

	// 第一次还没有开始更新
	if status.IsKanaryAnalysisRunning(&ka.Status) == false && len(t.Pods) == t.Replicas && t.HasTempWorkload == false {
		condition = appsv1alpha1.WorkloadNoUpdatKanaryAnalysisConditionType
		return &condition
	}

	// 说明验证已经完成
	if len(t.Pods) == t.Replicas && status.IsKanaryAnalysisRunning(&ka.Status) {
		condition = appsv1alpha1.WorkloadUpdatedKanaryAnalysisConditionType
		return &condition
	}

	// 正在验证过程中
	if len(t.Pods) > 0 && len(t.Pods) < t.Replicas && t.HasTempWorkload {
		condition = appsv1alpha1.RunningKanaryAnalysisConditionType
		return &condition
	}

	return &condition
}