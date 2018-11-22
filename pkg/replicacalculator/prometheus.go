package replicacalculator

import (
	"context"
	"fmt"
	"github.com/golang/glog"
	prometheusclient "github.com/prometheus/client_golang/api"
	prometheusapi "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"strings"
	"time"
)

const (
	prometheusQuery = `sum(rate(container_cpu_usage_seconds_total{pod_name=~"%s", namespace="%s"}[1m])) by(pod_name) / 
		sum(kube_pod_container_resource_requests_cpu_cores{pod_name=~"%s", namespace="%s"}) by (pod_name)`
)

type MetricsGetter interface {
	GetPodMetrics(namespace string, podIDs []string) (map[string][]int, error)
}

func NewPrometheusGetter(prometheusClient prometheusclient.Client) MetricsGetter {
	prometheusAPI := prometheusapi.NewAPI(prometheusClient)
	return &metricsGetter{prometheusClient: prometheusClient, prometheusAPI: prometheusAPI}
}

type metricsGetter struct {
	prometheusClient prometheusclient.Client
	prometheusAPI    prometheusapi.API
}

func (m *metricsGetter) GetPodMetrics(namespace string, podIDs []string) (map[string][]int, error) {
	todo := context.TODO()
	nameList := strings.Join(podIDs, "|")
	glog.Info(nameList)
	query := fmt.Sprintf(prometheusQuery, nameList, namespace, nameList, namespace)
	glog.Info(query)

	now := time.Now()
	end := now.Truncate(time.Minute)
	start := end.Add(-time.Minute * 5)
	queryRange := prometheusapi.Range{Start: start, End: end, Step: time.Minute}

	glog.Infof("query: %v", queryRange)
	results, err := m.prometheusAPI.QueryRange(todo, query, queryRange)
	if err != nil {
		return nil, err
	}
	var (
		matrixResult model.Matrix
		ok           bool
	)
	glog.Infof("results: %v", results)
	if matrixResult, ok = results.(model.Matrix); !ok {
		return nil, fmt.Errorf("unexpected return type from the prometheus api call: %v", results.Type())
	}
	mapResults := make(map[string][]int)
	for _, r := range matrixResult {
		podName := string(r.Metric["pod_name"])
		mapResults[podName] = make([]int, len(r.Values))
		for i, v := range r.Values {
			mapResults[podName][i] = int(v.Value * 100)
		}
	}
	return mapResults, nil
}
