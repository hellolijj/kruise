package validation

import (
	appsv1alpha1 "github.com/openkruise/kruise/pkg/apis/apps/v1alpha1"
	"github.com/openkruise/kruise/pkg/controller/kanaryanalysis/anomalydetector"
	"github.com/openkruise/kruise/pkg/controller/kanaryanalysis/utils/types"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

func NewPromQL(validationItems *appsv1alpha1.KanaryValidation, item *appsv1alpha1.KanaryValidationItem) ValidateStrategy {
	if validationItems.ValidationPeriod == nil {
		return &promQLImpl{
			validationSpec: *item.PromQL,
		}
	}

	return &promQLImpl{
		validationSpec:   *item.PromQL,
		validationPeriod: validationItems.ValidationPeriod.Duration,
	}
}

type promQLImpl struct {
	validationSpec   appsv1alpha1.KanaryValidationPromQL
	validationPeriod time.Duration

	anomalyDetector        anomalydetector.AnomalyDetector
	anomalyDetectorFactory anomalydetector.Factory
}

func (p *promQLImpl) initAnomalyDetector(client client.Client) error {
	anomalyDetectorConfig := anomalydetector.FactoryConfig{
		ValueInRangeConfig: &anomalydetector.ValueInRangeConfig{
			Min: p.validationSpec.ValueInRange.Min,
			Max: p.validationSpec.ValueInRange.Max,
		},
		PromConfig: &anomalydetector.ConfigPrometheusAnomalyDetector{
			PrometheusService: p.validationSpec.PrometheusService,
			Query:             p.validationSpec.Query,
		},
	}

	if p.anomalyDetectorFactory == nil {
		p.anomalyDetectorFactory = anomalydetector.New
	}

	var err error
	if p.anomalyDetector, err = p.anomalyDetectorFactory(anomalyDetectorConfig); err != nil {
		return err
	}

	return nil
}
func (p *promQLImpl) Validate(client client.Client, pods []*v1.Pod) (*types.ValidateResult, error) {
	var err error
	result := &types.ValidateResult{}

	if err = p.initAnomalyDetector(client); err != nil {
		return result, err
	}

	return p.anomalyDetector.CheckOutOfRange(pods)
}
