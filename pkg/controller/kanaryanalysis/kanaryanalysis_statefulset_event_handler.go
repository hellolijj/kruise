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
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ handler.EventHandler = &enqueueRequestForStatefulset{}

type enqueueRequestForStatefulset struct {
	client client.Client
}

func (s *enqueueRequestForStatefulset) Create(evt event.CreateEvent, q workqueue.RateLimitingInterface) {
	s.addStatefulset(q, evt.Object)
}

func (s *enqueueRequestForStatefulset) Delete(evt event.DeleteEvent, q workqueue.RateLimitingInterface) {
	s.deleteStatefulset(q, evt.Object)
}

func (s *enqueueRequestForStatefulset) Generic(evt event.GenericEvent, q workqueue.RateLimitingInterface) {
}

func (s *enqueueRequestForStatefulset) Update(evt event.UpdateEvent, q workqueue.RateLimitingInterface) {
	s.updateStatefulset(q, evt.ObjectOld, evt.ObjectNew)
}

// When a pod is added, figure out what sidecarSets it will be a member of and
// enqueue them. obj must have *v1.Pod type.
func (d *enqueueRequestForStatefulset) addStatefulset(q workqueue.RateLimitingInterface, obj runtime.Object) {
	sts, ok := obj.(*appsv1.StatefulSet)
	if !ok {
		return
	}

	kanaryAnalysises, err := d.getStatefulSetKanaryAnalysis(sts)
	if err != nil {
		klog.Errorf("unable to get kanary analysis related with deployment %s/%s, err: %v", sts.Namespace, sts.Name, err)
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

func (d *enqueueRequestForStatefulset) deleteStatefulset(q workqueue.RateLimitingInterface, obj runtime.Object) {
	if _, ok := obj.(*appsv1.StatefulSet); ok {
		d.addStatefulset(q, obj)
		return
	}
}

func (d *enqueueRequestForStatefulset) updateStatefulset(q workqueue.RateLimitingInterface, old, cur runtime.Object) {
	newStatefulset := cur.(*appsv1.StatefulSet)
	oldStatefulset := old.(*appsv1.StatefulSet)
	if newStatefulset.ResourceVersion == oldStatefulset.ResourceVersion {
		return
	}

	statefulsetChanged := isStatefulSetImageChanged(oldStatefulset, newStatefulset)

	if !statefulsetChanged {
		return
	}

	kanaryAnalysises, err := d.getStatefulsetKAMemberships(newStatefulset)
	if err != nil {
		klog.Errorf("unable to get sidecarSets of pod %s/%s, err: %v", newStatefulset.Namespace, newStatefulset.Name, err)
		return
	}

	for name := range kanaryAnalysises {
		q.Add(reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      name,
				Namespace: newStatefulset.GetNamespace(),
			},
		})
	}
}

func (s *enqueueRequestForStatefulset) getStatefulSetKanaryAnalysis(sts *appsv1.StatefulSet) ([]appsv1alpha1.KanaryAnalysis, error) {
	kanaryAnalysisList := appsv1alpha1.KanaryAnalysisList{}
	if err := s.client.List(context.TODO(), &client.ListOptions{}, &kanaryAnalysisList); err != nil {
		return nil, err
	}
	var matchKanaryAnalysis []appsv1alpha1.KanaryAnalysis
	for _, kanaryAnalysis := range kanaryAnalysisList.Items {
		matched, err := statefulsetMatchKanaryAnalysis(sts, kanaryAnalysis)
		if err != nil {
			return nil, err
		}
		if matched {
			matchKanaryAnalysis = append(matchKanaryAnalysis, kanaryAnalysis)
		}
	}
	return matchKanaryAnalysis, nil
}

func (d *enqueueRequestForStatefulset) getStatefulsetKAMemberships(sts *appsv1.StatefulSet) (sets.String, error) {
	set := sets.String{}
	sidecarSets, err := d.getStatefulSetKanaryAnalysis(sts)
	if err != nil {
		return set, err
	}

	for _, sidecarSet := range sidecarSets {
		set.Insert(sidecarSet.Name)
	}
	return set, nil
}

func statefulsetMatchKanaryAnalysis(s *appsv1.StatefulSet, ka appsv1alpha1.KanaryAnalysis) (bool, error) {
	workload := ka.Spec.Workload
	if &workload == nil || s == nil {
		return false, fmt.Errorf("workload (%v) deployment (%v) can't be nil. ", workload, s)
	}
	// api name ns version都相等就行
	if workload.Name == s.Name && workload.APIVersion == "apps/v1" && workload.Kind == "deployment" {
		return true, nil
	}
	return false, nil
}

func isStatefulSetImageChanged(oldS, newS *appsv1.StatefulSet) bool {
	// check image
	oldContainers := oldS.Spec.Template.Spec.Containers
	newContainers := newS.Spec.Template.Spec.Containers

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
