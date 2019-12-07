package validation

import (
	appsv1alpha1 "github.com/openkruise/kruise/pkg/apis/apps/v1alpha1"
	"github.com/openkruise/kruise/pkg/controller/kanaryanalysis/utils/types"
	v1 "k8s.io/api/core/v1"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Interface interface {
	Apply(client client.Client, analysis *appsv1alpha1.KanaryAnalysis, pods []*v1.Pod) (*types.ValidateResult, error)
}

type strategy struct {
	validations []ValidateStrategy
}

type ValidateStrategy interface {
	Validate(client client.Client, pods []*v1.Pod) (*types.ValidateResult, error)
}

func New(spec *appsv1alpha1.KanaryAnalysisSpec) (Interface, error) {
	var validationsImpls []ValidateStrategy
	for _, v := range spec.Validation.Items {
		if v.PromQL != nil {
			validationsImpls = append(validationsImpls, NewPromQL(&spec.Validation, &v))
		}
		// todo add other validaion
	}
	return &strategy{validations: validationsImpls}, nil
}

func (s *strategy) Apply(client client.Client, ka *appsv1alpha1.KanaryAnalysis, pods []*v1.Pod) (*types.ValidateResult, error) {
	for _, validation := range s.validations {
		result, err := validation.Validate(client, pods)
		if err != nil {
			klog.Errorf("validation err: %v", err)
			return result, err
		}
		if result != nil && result.IsFailed {
			return result, nil
		}
	}
	return nil, nil
}
