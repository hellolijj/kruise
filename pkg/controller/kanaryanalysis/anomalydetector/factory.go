package anomalydetector

import (
	"fmt"
	"github.com/openkruise/kruise/pkg/controller/kanaryanalysis/utils/types"
	v1 "k8s.io/api/core/v1"
)

type AnomalyDetector interface {
	CheckOutOfRange(pods []*v1.Pod) (*types.ValidateResult, error)
}

type FactoryConfig struct {
	ValueInRangeConfig *ValueInRangeConfig
	PromConfig         *ConfigPrometheusAnomalyDetector
}

type Factory func(cfg FactoryConfig) (AnomalyDetector, error)

var _ Factory = New

// todo: add no config case
func New(cfg FactoryConfig) (AnomalyDetector, error) {

	if cfg.PromConfig != nil && cfg.ValueInRangeConfig != nil {
		return newValueInRangeWithProm(*cfg.ValueInRangeConfig, *cfg.PromConfig)
	}
	return nil, fmt.Errorf("invalid multiple configuration: %v", cfg)
}

func newValueInRangeWithProm(configValueInRange ValueInRangeConfig, configProm ConfigPrometheusAnomalyDetector) (AnomalyDetector, error) {
	a := &ValueInRangeAnalyser{
		ConfigSpecific: configValueInRange,
	}

	var err error
	if a.analyser, err = newPromValueInRangeAnalyser(configProm, configValueInRange); err != nil {
		return nil, err
	}
	return a, nil
}
