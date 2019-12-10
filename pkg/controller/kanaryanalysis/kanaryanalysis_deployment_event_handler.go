package kanaryanalysis

import (
	"context"
	"fmt"
	appsv1alpha1 "github.com/openkruise/kruise/pkg/apis/apps/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"
	"k8s.io/kubernetes/pkg/apis/apps"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ handler.EventHandler = &enqueueRequestForDeployment{}

type enqueueRequestForDeployment struct {
	client client.Client
}

func (d *enqueueRequestForDeployment) Create(evt event.CreateEvent, q workqueue.RateLimitingInterface) {
	d.addDeployment(q, evt.Object)
}

func (d *enqueueRequestForDeployment) Delete(evt event.DeleteEvent, q workqueue.RateLimitingInterface) {
	d.deleteDeployment(q, evt.Object)
}

func (d *enqueueRequestForDeployment) Generic(evt event.GenericEvent, q workqueue.RateLimitingInterface) {
}

func (d *enqueueRequestForDeployment) Update(evt event.UpdateEvent, q workqueue.RateLimitingInterface) {
	d.updateDeployment(q, evt.ObjectOld, evt.ObjectNew)
}

// When a pod is added, figure out what sidecarSets it will be a member of and
// enqueue them. obj must have *v1.Pod type.
func (d *enqueueRequestForDeployment) addDeployment(q workqueue.RateLimitingInterface, obj runtime.Object) {
	deployment, ok := obj.(*appsv1.Deployment)
	if !ok {
		return
	}

	kanaryAnalysises, err := d.getDeploymentKanaryAnalysis(deployment)
	if err != nil {
		klog.Errorf("unable to get kanary analysis related with deployment %s/%s, err: %v", deployment.Namespace, deployment.Name, err)
		return
	}
	for _, ka := range kanaryAnalysises {
		q.Add(reconcile.Request{
			NamespacedName: types.NamespacedName{
				Namespace: ka.GetNamespace(),
				Name:      ka.GetName(),
			},
		})
	}
}

func (d *enqueueRequestForDeployment) deleteDeployment(q workqueue.RateLimitingInterface, obj runtime.Object) {
	if _, ok := obj.(*apps.Deployment); ok {
		d.addDeployment(q, obj)
		return
	}
}

func (d *enqueueRequestForDeployment) updateDeployment(q workqueue.RateLimitingInterface, old, cur runtime.Object) {
	newDeployment := cur.(*appsv1.Deployment)
	oldDeployment := old.(*appsv1.Deployment)
	klog.Infof("new deploy reversion: %v old deploy reversion %v", newDeployment.ResourceVersion, oldDeployment.ResourceVersion)
	if newDeployment.ResourceVersion == oldDeployment.ResourceVersion {
		return
	}

	isDeploymentImageChanged := isDeploymentImageChanged(oldDeployment, newDeployment)
	if !isDeploymentImageChanged {
		return
	}
	klog.Infof("isDeploymentImageChanged %v", isDeploymentImageChanged)

	kanaryAnalysises, err := d.getDeploymentKAMemberships(newDeployment)
	klog.Infof(" deploy kanaryAnalysises %v", kanaryAnalysises)
	if err != nil {
		klog.Errorf("unable to get sidecarSets of pod %s/%s, err: %v", newDeployment.Namespace, newDeployment.Name, err)
		return
	}

	for name := range kanaryAnalysises {
		q.Add(reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      name,
				Namespace: newDeployment.GetNamespace(),
			},
		})
	}
}

func (d *enqueueRequestForDeployment) getDeploymentKanaryAnalysis(deployment *appsv1.Deployment) ([]appsv1alpha1.KanaryAnalysis, error) {
	kanaryAnalysisList := appsv1alpha1.KanaryAnalysisList{}
	if err := d.client.List(context.TODO(), &client.ListOptions{}, &kanaryAnalysisList); err != nil {
		return nil, err
	}
	var matchKanaryAnalysis []appsv1alpha1.KanaryAnalysis
	for _, kanaryAnalysis := range kanaryAnalysisList.Items {
		matched, err := deploymentMatchKanaryAnalysis(deployment, kanaryAnalysis)
		if err != nil {
			return nil, err
		}
		if matched {
			matchKanaryAnalysis = append(matchKanaryAnalysis, kanaryAnalysis)
		}
	}
	return matchKanaryAnalysis, nil
}

func (d *enqueueRequestForDeployment) getDeploymentKAMemberships(deployment *appsv1.Deployment) (sets.String, error) {
	set := sets.String{}
	kanaryAnalysises, err := d.getDeploymentKanaryAnalysis(deployment)
	if err != nil {
		return set, err
	}

	for _, ka := range kanaryAnalysises {
		set.Insert(ka.Name)
	}
	return set, nil
}

func deploymentMatchKanaryAnalysis(d *appsv1.Deployment, ka appsv1alpha1.KanaryAnalysis) (bool, error) {
	workload := ka.Spec.Workload
	if &workload == nil || d == nil {
		return false, fmt.Errorf("workload (%v) deployment (%v) can't be nil. ", workload, d)
	}
	// api name ns version都相等就行
	if workload.Name == d.Name && workload.APIVersion == "apps/v1" && workload.Kind == "deployment" {
		return true, nil
	}
	return false, nil
}

func isDeploymentImageChanged(oldD, newD *appsv1.Deployment) bool {
	// check image
	oldContainers := oldD.Spec.Template.Spec.Containers
	newContainers := newD.Spec.Template.Spec.Containers

	if len(oldContainers) != len(newContainers) {
		return true
	}

	for i := 0; i < len(oldContainers); i++ {
		if oldContainers[i].Image != newContainers[i].Image {
			return true
		}
	}
	return false
}
