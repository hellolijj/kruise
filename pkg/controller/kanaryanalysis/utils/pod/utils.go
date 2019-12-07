package pod

import v1 "k8s.io/api/core/v1"

//FilterOut remove from the slice pods for which exclude function return true
func FilterOut(slice []*v1.Pod, exclude func(*v1.Pod) (bool, error)) ([]*v1.Pod, error) {
	b := []*v1.Pod{}
	for _, x := range slice {
		ok, err := exclude(x)
		if err != nil {
			return b, err
		}
		if !ok {
			b = append(b, x)
		}
	}
	return b, nil
}

//PurgeNotReadyPods keep only pods that are ready inside the slice
func PurgeNotReadyPods(pods []*v1.Pod) ([]*v1.Pod, error) {
	return FilterOut(pods, func(a *v1.Pod) (bool, error) { return !IsReady(a), nil })
}

//IsReady check if the pod is Ready
func IsReady(p *v1.Pod) bool {
	if p.Status.Phase == v1.PodRunning {
		for _, c := range p.Status.Conditions {
			if c.Type == v1.PodReady {
				return c.Status == v1.ConditionTrue
			}
		}
	}
	return false
}
