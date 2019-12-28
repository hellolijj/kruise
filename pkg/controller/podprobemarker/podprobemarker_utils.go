package podprobemarker

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/kubectl/util/podutils"
)

func isRunningAndReady(pod *v1.Pod) bool {
	return pod.Status.Phase == v1.PodRunning && podutils.IsPodReady(pod)
}
