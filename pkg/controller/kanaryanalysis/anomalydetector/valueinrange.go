package anomalydetector

import (
	"fmt"
	"github.com/openkruise/kruise/pkg/controller/kanaryanalysis/utils/pod"
	"github.com/openkruise/kruise/pkg/controller/kanaryanalysis/utils/types"
	v1 "k8s.io/api/core/v1"
)

type ValueInRangeConfig struct {
	Min *float64
	Max *float64
}

type ValueInRangeAnalyser struct {
	ConfigSpecific ValueInRangeConfig
	analyser       valueInRangeAnalyser
}

type valueInRangeAnalyser interface {
	doAnalysis(pods []*v1.Pod) (inRangeByPodName, error)
}

type inRangeByPodName map[string]float64

func (d *ValueInRangeAnalyser) CheckOutOfRange(pods []*v1.Pod) (*types.ValidateResult, error) {
	if len(pods) == 0 {
		return nil, fmt.Errorf("no pod to check")
	}

	pods, err := pod.PurgeNotReadyPods(pods)
	if err != nil {
		return nil, fmt.Errorf("can't purge not ready pods, err: %v", err)
	}

	podAnalysisResult, err := d.analyser.doAnalysis(pods)
	if err != nil {
		return nil, err
	}

	for pod, value := range podAnalysisResult {
		if value < *d.ConfigSpecific.Min || value > *d.ConfigSpecific.Max {
			return &types.ValidateResult{
				IsFailed: true,
				Reason:   fmt.Sprintf("pod %s value %.2f is not betwwen range %.2f ~ %.2f", pod, value, *d.ConfigSpecific.Min, *d.ConfigSpecific.Max),
			}, nil
		}
	}

	return nil, nil
}
