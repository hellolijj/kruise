package workloadpods

import (
	"context"
	"fmt"
	"github.com/openkruise/kruise/pkg/apis/apps/v1alpha1"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/kubernetes/pkg/controller/deployment/util"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type disOrder struct {
	name       string
	namespaces string
}

func (d *disOrder) ListToCheckPods(clientSet client.Client, ka *v1alpha1.KanaryAnalysis) ([]*corev1.Pod, int, error) {
	deployment := &v1.Deployment{}
	if err := clientSet.Get(context.TODO(), types.NamespacedName{
		Namespace: d.namespaces,
		Name:      d.name,
	}, deployment); err != nil {
		return nil, -1, err
	}

	replicaSetList := &v1.ReplicaSetList{}
	selector, err := metav1.LabelSelectorAsSelector(deployment.Spec.Selector)
	if err != nil {
		return nil, -1, err
	}
	if err := clientSet.List(context.TODO(), &client.ListOptions{LabelSelector: selector}, replicaSetList); err != nil {
		return nil, -1, err
	}
	var replicaSets []*v1.ReplicaSet
	for i := range replicaSetList.Items {
		replicaSets = append(replicaSets, &replicaSetList.Items[i])
	}

	newRS := util.FindNewReplicaSet(deployment, replicaSets)
	if newRS == nil {
		return nil, -1, fmt.Errorf("get replicas set is nil")
	}

	podList := &corev1.PodList{}
	selector = labels.SelectorFromSet(labels.Set{"pod-template-hash": newRS.Labels["pod-template-hash"]})
	if err := clientSet.List(context.TODO(), &client.ListOptions{LabelSelector: selector}, podList); err != nil {
		return nil, -1, fmt.Errorf("failed to get podList from replicaSet %s", newRS)
	}

	var pods []*corev1.Pod
	for _, pod := range podList.Items {
		if pod.Status.Phase == corev1.PodRunning {
			pods = append(pods, &pod)
		}
	}

	return pods, int(deployment.Status.Replicas), nil
}

func (d *disOrder) SetPausedTrue(clientSet client.Client) error {
	oldDeployment := &v1.Deployment{}
	if err := clientSet.Get(context.TODO(), types.NamespacedName{
		Namespace: d.namespaces,
		Name:      d.name,
	}, oldDeployment); err != nil {
		return err
	}

	newDeployment := oldDeployment.DeepCopy()
	newDeployment.Spec.Paused = true
	return clientSet.Update(context.TODO(), newDeployment)
}

func (d *disOrder) String() string {
	return d.namespaces + "/" + d.name
}
