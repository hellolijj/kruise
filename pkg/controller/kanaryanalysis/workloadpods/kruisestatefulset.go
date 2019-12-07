package workloadpods

import  (
	"context"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	types2 "github.com/openkruise/kruise/pkg/controller/kanaryanalysis/utils/types"


	"k8s.io/klog"

	"github.com/openkruise/kruise/pkg/apis/apps/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type kruiseOrder struct {
	workloadName string
	namespaces   string
}

func (k *kruiseOrder) ListToCheckPods(clientSet client.Client, ka *v1alpha1.KanaryAnalysis) (*types2.ToCheckPods, error) {
	statefulset := &v1alpha1.StatefulSet{}
	if err := clientSet.Get(context.TODO(), types.NamespacedName{
		Namespace: k.namespaces,
		Name:      k.workloadName,
	}, statefulset); err != nil {
		return nil, err
	}

	klog.Info("get kruise statefulset:", statefulset.Name)

	podList := &v1.PodList{}
	if err := clientSet.List(context.TODO(), &client.ListOptions{
		LabelSelector: labels.SelectorFromSet(labels.Set{"controller-revision-hash": statefulset.Status.UpdateRevision}),
	}, podList); err != nil {
		return nil, err
	}

	var pods []*v1.Pod
	for i := range podList.Items {
		pods = append(pods, &podList.Items[i])
	}

	hasTempWorkload := false
	if statefulset.Status.UpdateRevision != statefulset.Status.CurrentRevision {
		hasTempWorkload = true
	}

	return &types2.ToCheckPods{
		Pods:     pods,
		Replicas: int(statefulset.Status.Replicas),
		HasTempWorkload: hasTempWorkload,
	}, nil
}

func (k *kruiseOrder) SetPausedTrue(clientSet client.Client) error {
	oldStatefulSet := &v1alpha1.StatefulSet{}
	if err := clientSet.Get(context.TODO(), types.NamespacedName{
		Namespace: k.namespaces,
		Name:      k.workloadName,
	}, oldStatefulSet); err != nil {
		return err
	}

	newStatefulSet := oldStatefulSet.DeepCopy()
	newStatefulSet.Spec.UpdateStrategy.RollingUpdate.Paused = true
	return clientSet.Update(context.TODO(), newStatefulSet)
}

func (k *kruiseOrder) String() string {
	return k.namespaces + "/" + k.workloadName
}