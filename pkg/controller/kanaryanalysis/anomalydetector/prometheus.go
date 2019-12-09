package anomalydetector

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	promClient "github.com/prometheus/client_golang/api"
	promApi "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	v1 "k8s.io/api/core/v1"
	"k8s.io/klog"
	"time"
)

//ConfigPrometheusAnomalyDetector configuration to connect to prometheus
type ConfigPrometheusAnomalyDetector struct {
	PrometheusService string
	Query             string

	queryAPI promApi.API
	logger   logr.Logger
}

// ====== ValueInRangeAnalyser ========

type promValueRangeAnalyser struct {
	promConfig ConfigPrometheusAnomalyDetector
	config     ValueInRangeConfig
}

func newPromValueInRangeAnalyser(promConfig ConfigPrometheusAnomalyDetector, config ValueInRangeConfig) (*promValueRangeAnalyser, error) {
	promInitConfig := promClient.Config{
		Address: "http://" + promConfig.PrometheusService,
	}
	prometheusClient, err := promClient.NewClient(promInitConfig)
	if err != nil {
		return nil, err
	}
	promConfig.queryAPI = promApi.NewAPI(prometheusClient)

	return &promValueRangeAnalyser{
		promConfig: promConfig,
		config:     config,
	}, nil
}

// true 表示在范围之内。 false 表示不在范围之内
func (p *promValueRangeAnalyser) doAnalysis(pods []*v1.Pod) (inRangeByPodName, error) {
	ctx := context.Background()
	tsNow := time.Now()
	m, err := p.promConfig.queryAPI.Query(ctx, p.promConfig.Query, tsNow)

	if err != nil {
		return nil, fmt.Errorf("error processing prometheus query: %s", err)
	}
	if m == nil {
		return nil, fmt.Errorf("error query result: %s", err)
	}

	vector, ok := m.(model.Vector)
	if !ok {
		return nil, fmt.Errorf("the prometheus query did not return a result in the form of expected types 'model.Vector': %s", err)
	}

	allSearchResult := inRangeByPodName{}
	for _, sample := range vector {
		podName := string(sample.Metric["pod"])
		allSearchResult[podName] = float64(sample.Value)
	}

	matchResult := inRangeByPodName{}
	for _, pod := range pods {
		if value, ok := allSearchResult[pod.Name]; ok {
			matchResult[pod.Name] = value
		}
	}

	klog.V(4).Infof("prometheus search ql: %v, all result %v, match result %v ", p.promConfig, allSearchResult, matchResult)

	return matchResult, nil
}
