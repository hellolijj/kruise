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
		OwnerType:    &appsv1alpha1.KanaryAnalysis{},
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
		status.UpdateKanaryAnalysisStatus(r.Client, ka, appsv1alpha1.ErroredKanaryAnalysisConditionType, "workload init error")
		return reconcile.Result{}, fmt.Errorf("workload match nil")
	}
	klog.V(4).Infof("get workload: %s", workload.String())
	pods, replicas, err := workload.ListToCheckPods(r.Client, ka)
	if err != nil {
		return nextReconcile(ka, status.UpdateKanaryAnalysisStatus(r.Client, ka, appsv1alpha1.ErroredKanaryAnalysisConditionType, err.Error()))
	}
	klog.V(4).Infof("get workload updated pods count: %d", len(pods))

	r.updateKanaryAnalysisStatus(ka, len(pods), replicas)
	if isContinue, res, err := r.isContinueByKAStatus(ka); !isContinue {
		return res, err
	}

	// 验证 pod....
	validationStrategy, err := validation.New(&ka.Spec)
	if err != nil {
		status.UpdateKanaryAnalysisStatus(r.Client, ka, appsv1alpha1.ErroredKanaryAnalysisConditionType, err.Error())
		return reconcile.Result{}, err
	}
	res, err := validationStrategy.Apply(r.Client, ka, pods)
	if err != nil {
		status.UpdateKanaryAnalysisStatus(r.Client, ka, appsv1alpha1.ErroredKanaryAnalysisConditionType, err.Error())
		return reconcile.Result{}, err
	}
	klog.V(4).Infof("get validation result: %v", res)

	if res == nil {
		err = fmt.Errorf("validation result is nil")
		status.UpdateKanaryAnalysisStatus(r.Client, ka, appsv1alpha1.ErroredKanaryAnalysisConditionType, err.Error())
		return reconcile.Result{}, err
	}

	// 验证失败，喊停, 验证成功，持续验证
	if res.IsFailed {
		status.UpdateKanaryAnalysisStatus(r.Client, ka, appsv1alpha1.FailedKanaryAnalysisConditionType, res.Reason)
		return reconcile.Result{}, nil
	} else {
		status.UpdateKanaryAnalysisStatus(r.Client, ka, appsv1alpha1.SucceedKanaryAnalysisConditionType, res.Reason)
		return nextReconcile(ka, nil)
	}
}

func nextReconcile(ka *appsv1alpha1.KanaryAnalysis, err error) (reconcile.Result, error) {
	if ka.Spec.Validation.Interval != nil {
		return reconcile.Result{
			Requeue:      true,
			RequeueAfter: ka.Spec.Validation.Interval.Duration,
		}, err
	}
	// 默认 3s interval 的间隔
	return reconcile.Result{Requeue: true, RequeueAfter: 3 * time.Second}, err
}

// 根据 pods、replicas 数量更新 ka 状态
func (r *ReconcileKanaryAnalysis) updateKanaryAnalysisStatus(ka *appsv1alpha1.KanaryAnalysis, dPods, replicas int) {
	if dPods == 0 || replicas == 0 {
		return
	}

	// set deploy
	if dPods == replicas && status.IsKanaryAnalysisSucceed(&ka.Status) {
		status.UpdateKanaryAnalysisStatus(r.Client, ka, appsv1alpha1.DeployedKanryAnalysisConditionType, "Deployed pods finished")
		return
	}

	// set deploy
	// pod 没有验证完成，但是超过了验证时间段
	if status.IsKanaryAnalysisSucceed(&ka.Status) && validation.IsDeadlinePeriodDone(ka) {
		status.UpdateKanaryAnalysisStatus(r.Client, ka, appsv1alpha1.DeployedKanryAnalysisConditionType, "Exceed valid period")
		return
	}

	// set running
	if dPods != replicas {
		status.UpdateKanaryAnalysisStatus(r.Client, ka, appsv1alpha1.RunningKanaryAnalysisConditionType, "")
		return
	}
	// set ready
	if dPods == replicas && !status.IsKanaryAnalysisRunning(&ka.Status) {
		status.UpdateKanaryAnalysisStatus(r.Client, ka, appsv1alpha1.ReadyKanaryAnalysisConditionType, "")
		return
	}

	return
}

// 根据 ka 的 status 决定接下来的操作
// true 表示要继续往下执行
func (r *ReconcileKanaryAnalysis) isContinueByKAStatus(ka *appsv1alpha1.KanaryAnalysis) (bool, reconcile.Result, error) {

	if status.IsKanaryAnalysisFailed(&ka.Status) {
		klog.V(4).Infof("ka failed: %s", status.GetKanaryAnalysisFailedReason(&ka.Status))
		return false, reconcile.Result{}, nil
	}

	if status.IsKanaryAnalysisDeployed(&ka.Status) {
		klog.V(4).Infof("workload %s deployed", ka.Spec.Workload.Name)
		return false, reconcile.Result{}, nil
	}

	if status.IsKanaryAnalysisSucceed(&ka.Status) {
		return true, reconcile.Result{}, nil
	}

	if status.IsKanaryAnalysisRunning(&ka.Status) {
		return true, reconcile.Result{}, nil
	}

	if status.IsKanaryAnalysisErrored(&ka.Status) {
		return true, reconcile.Result{}, nil
	}

	if status.IsKanaryAnalysisReady(&ka.Status) {
		res, err := nextReconcile(ka, nil)
		return false, res, err
	}

	return false, reconcile.Result{}, nil
}

// 根据 Ka 状态是否进一步执行

// todo 统一灰度集成
// todo watch 资源过滤
// todo 队列里面 放 workload, name 通过分隔符判断 | workload

// todo 各种字段也用起来 还有 label watch and 手动获取没有实现。
