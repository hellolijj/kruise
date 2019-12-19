package rolloutdefinition

import (
	"context"
	"github.com/openkruise/kruise/pkg/apis/apps/v1alpha1"
	"k8s.io/apimachinery/pkg/types"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/util/workqueue"

	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type enqueueRolloutControlForDefinition struct {
	client client.Client
}

// 只有这个definition 的 apiversion 与 rolloutcontrl 的meta数据相互匹配的时候才放入队列
func (d *enqueueRolloutControlForDefinition) Create(evt event.CreateEvent, q workqueue.RateLimitingInterface) {
	d.addDefinition(q, evt.Object)
}

func (d *enqueueRolloutControlForDefinition) Delete(evt event.DeleteEvent, q workqueue.RateLimitingInterface) {
	d.deleteDefinition(q, evt.Object)
}

func (d *enqueueRolloutControlForDefinition) Generic(evt event.GenericEvent, q workqueue.RateLimitingInterface) {
}

func (d *enqueueRolloutControlForDefinition) Update(evt event.UpdateEvent, q workqueue.RateLimitingInterface) {
	d.updateDefinition(q, evt.ObjectOld, evt.ObjectNew)
}

func (d *enqueueRolloutControlForDefinition) addDefinition(q workqueue.RateLimitingInterface, obj runtime.Object) {
	unstructuredObj, ok := obj.(*unstructured.Unstructured)
	if !ok {
		return
	}

	rolloutControlList := &v1alpha1.RolloutControlList{}
	err := d.client.List(context.TODO(), &client.ListOptions{}, rolloutControlList)
	if err != nil {
		klog.Errorf("Error enqueueing rolloutControlList on addNode %v", err)
	}

	for _, rolloutControl := range rolloutControlList.Items {
		if unstructuredObj.GetAPIVersion() == rolloutControl.Spec.Resource.APIVersion && unstructuredObj.GetKind() == rolloutControl.Spec.Resource.Kind {
			q.Add(reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: rolloutControl.Namespace,
					Name:      rolloutControl.Name}})
		}
	}
}

func (p *enqueueRolloutControlForDefinition) deleteDefinition(q workqueue.RateLimitingInterface, obj runtime.Object) {
	unstructuredObj, ok := obj.(*unstructured.Unstructured)
	if !ok {
		return
	}
	p.addDefinition(q, unstructuredObj)
	return
}

func (d *enqueueRolloutControlForDefinition) updateDefinition(q workqueue.RateLimitingInterface, old, cur runtime.Object) {
	newObj := cur.(*unstructured.Unstructured)
	oldObj := old.(*unstructured.Unstructured)
	if newObj.GetResourceVersion() == oldObj.GetResourceVersion() {

		return
	}

	rolloutControlList := &v1alpha1.RolloutControlList{}
	err := d.client.List(context.TODO(), &client.ListOptions{}, rolloutControlList)
	if err != nil {
		klog.Errorf("Error enqueueing rolloutControlList on addNode %v", err)
	}

	for _, rolloutControl := range rolloutControlList.Items {
		if newObj.GetAPIVersion() == rolloutControl.Spec.Resource.APIVersion && newObj.GetKind() == rolloutControl.Spec.Resource.Kind {
			q.Add(reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: rolloutControl.Namespace,
					Name:      rolloutControl.Name}})
		}
	}
}
