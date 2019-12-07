package workloadpods

import (
	"context"
	"github.com/openkruise/kruise/pkg/apis/apps/v1alpha1"
	apps "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type order struct {
	workloadName string
	namespaces   string
}

func (o *order) ListToCheckPods(clientSet client.Client, ka *v1alpha1.KanaryAnalysis) ([]*v1.Pod, *v1alpha1.KanaryAnalysisConditionType, error) {
	statefulset := &apps.StatefulSet{}
	if err := clientSet.Get(context.TODO(), types.NamespacedName{
		Namespace: o.namespaces,
		Name:      o.workloadName,
	}, statefulset); err != nil {
		return nil, nil, err
	}

	klog.Info("get statefulset:", statefulset.Name)

	podList := &v1.PodList{}
	if err := clientSet.List(context.TODO(), &client.ListOptions{
		LabelSelector: labels.SelectorFromSet(labels.Set{"controller-revision-hash": statefulset.Status.UpdateRevision}),
	}, podList); err != nil {
		klog.Errorf("failed to get podList for workload %s", o.workloadName)
	}

	var pods []*v1.Pod
	for i := range podList.Items {
		pods = append(pods, &podList.Items[i])
	}

	status := v1alpha1.RunningKanaryAnalysisConditionType
	if statefulset.Status.CurrentRevision != statefulset.Status.UpdateRevision && len(pods) == int(statefulset.Status.Replicas) {
		status = v1alpha1.WorkloadUpdatedKanaryAnalysisConditionType
		return pods, &status, nil

	}
	return pods, &status, nil
}

// apps.statefulset has no pause field
func (o *order) SetPausedTrue(clientSet client.Client) error {
	return nil
}

func (o *order) String() string {
	return o.namespaces + "/" + o.workloadName
}