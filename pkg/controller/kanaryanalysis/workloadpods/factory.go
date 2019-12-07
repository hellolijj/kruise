package workloadpods

import (
	"github.com/openkruise/kruise/pkg/apis/apps/v1alpha1"
	"github.com/openkruise/kruise/pkg/controller/kanaryanalysis/utils/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Interface interface {
	ListToCheckPods(client client.Client, ka *v1alpha1.KanaryAnalysis) (*types.ToCheckPods, error)
	SetPausedTrue(client client.Client) error
	String() string

}

func New(spec *v1alpha1.KanaryAnalysisSpec) Interface {
	if spec.SelectedDeployedPodMethod == v1alpha1.Disordered && spec.Workload.Kind == "deployment" {
		return &disOrder{name: spec.Workload.Name, namespaces: spec.Workload.Namespace}
	}

	if spec.SelectedDeployedPodMethod == v1alpha1.Ordered && spec.Workload.Kind == "statefulset" && spec.Workload.APIVersion == "apps.kruise.io" {
		return &kruiseOrder{workloadName: spec.Workload.Name, namespaces: spec.Workload.Namespace}
	}
	//
	//if spec.SelectedDeployedPodMethod == v1alpha1.Ordered && spec.Workload.Kind == "statefulset" {
	//	return &order{workloadName: spec.Workload.Name, namespaces: spec.Workload.Namespace}
	//}

	return nil
}
