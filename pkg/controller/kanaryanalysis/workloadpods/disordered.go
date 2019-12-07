package workloadpods

import (
	"context"
	"fmt"
	"github.com/openkruise/kruise/pkg/apis"
	"github.com/openkruise/kruise/pkg/apis/apps/v1alpha1"
	"k8s.io/klog"

	types2 "github.com/openkruise/kruise/pkg/controller/kanaryanalysis/utils/types"


	"k8s.io/apimachinery/pkg/runtime"

	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

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


func (d *disOrder) ListToCheckPods(clientSet client.Client, ka *v1alpha1.KanaryAnalysis) (*types2.ToCheckPods, error) {
	deployment := &v1.Deployment{}
	if err := clientSet.Get(context.TODO(), types.NamespacedName{
		Namespace: d.namespaces,
		Name:      d.name,
	}, deployment); err != nil {
		return nil, err
	}

	// set own tome
	//err := addOwnerReferences(clientSet, ka, deployment)
	//if err != nil {
	//	return nil, err
	//}

	replicaSetList := &v1.ReplicaSetList{}
	selector, err := metav1.LabelSelectorAsSelector(deployment.Spec.Selector)
	if err != nil {
		return nil ,err
	}
	if err := clientSet.List(context.TODO(), &client.ListOptions{LabelSelector: selector}, replicaSetList); err != nil {
		return nil, err
	}
	var replicaSets []*v1.ReplicaSet
	for i := range replicaSetList.Items {
		replicaSets = append(replicaSets, &replicaSetList.Items[i])
	}

	newRS := util.FindNewReplicaSet(deployment, replicaSets)
	if newRS == nil {
		return nil, fmt.Errorf("get replicas set is nil")
	}

	podList := &corev1.PodList{}
	selector = labels.SelectorFromSet(labels.Set{"pod-template-hash": newRS.Labels["pod-template-hash"]})
	if err := clientSet.List(context.TODO(), &client.ListOptions{LabelSelector: selector}, podList); err != nil {
		return nil, fmt.Errorf("failed to get podList from replicaSet %s", newRS)
	}

	var pods []*corev1.Pod
	for _, pod := range podList.Items {
		if pod.Status.Phase == corev1.PodRunning {
			pods = append(pods, &pod)
		}
	}

	oldRSs, _ := util.FindOldReplicaSets(deployment, replicaSets)

	hasTempWorkload := false
	if len(oldRSs) > 0 {
		klog.V(4).Infof("old rs count %d", len(oldRSs))
		hasTempWorkload = true
	}

	return &types2.ToCheckPods{
		Pods:     pods,
		Replicas: int(deployment.Status.Replicas),
		HasTempWorkload: hasTempWorkload,
	}, nil
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

//PrepareSchemeForOwnerRef return the scheme required to write the kanary ownerreference
func PrepareSchemeForOwnerRef() *runtime.Scheme {
	scheme := runtime.NewScheme()
	if err := apis.AddToScheme(scheme); err != nil {
		panic(err.Error())
	}
	return scheme
}

func addOwnerReferences(clientSet client.Client, ka *v1alpha1.KanaryAnalysis, deployment *v1.Deployment) error {

	dpOwners := deployment.OwnerReferences
	isOwnerKa := false
	for _, dpOwner := range dpOwners {
		if dpOwner.Kind == ka.Kind && dpOwner.Name == ka.Name {
			isOwnerKa = true
			break
		}
	}
	if isOwnerKa == false {
		newDp := deployment.DeepCopy()
		err := controllerutil.SetControllerReference(ka, newDp,PrepareSchemeForOwnerRef())
		if err != nil {
			return err
		}
		return clientSet.Update(context.TODO(), newDp)
	}

	return nil
}


// // FindOldReplicaSets returns the old replica sets targeted by the given Deployment, with the given slice of RSes.
// // Note that the first set of old replica sets doesn't include the ones with no pods, and the second set of old replica sets include all old replica sets.
func findOldReplicaSets(deployment *v1.Deployment, rsList []*v1.ReplicaSet, newRS *v1.ReplicaSet) ([]*v1.ReplicaSet, []*v1.ReplicaSet) {
	var requiredRSs []*v1.ReplicaSet
	var allRSs []*v1.ReplicaSet
	for _, rs := range rsList {
		// Filter out new replica set
		if newRS != nil && rs.UID == newRS.UID {
			continue
		}
		allRSs = append(allRSs, rs)
		if *(rs.Spec.Replicas) != 0 {
			requiredRSs = append(requiredRSs, rs)
		}
	}
	return requiredRSs, allRSs
}