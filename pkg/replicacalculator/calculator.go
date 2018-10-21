package replicacalculator

import (
	"github.com/golang/glog"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/kubernetes/pkg/controller/podautoscaler/metrics"
	"time"
)

type ReplicaCalculator struct {
	metricsClient metrics.MetricsClient
	podLister     corelisters.PodLister
}

// NewReplicaCalculator Creates a  new replica calculator
func NewReplicaCalculator(client metrics.MetricsClient, lister corelisters.PodLister) *ReplicaCalculator {
	return &ReplicaCalculator{
		metricsClient: client,
		podLister:     lister,
	}
}

// GetResourceReplicas get number of replicas for the deployment
func (c *ReplicaCalculator) GetResourceReplicas(currentReplicas int32, downThreshold int32, upThreshold int32, resource v1.ResourceName,
	namespace string, selector labels.Selector) (int32, int32, int64, time.Time, error) {
	metrics, timestamp, err := c.metricsClient.GetResourceMetric(resource, namespace, selector)
	if err != nil {
		return 0, 0, 0, time.Time{}, err
	}
	glog.Infof("metrics: %v", metrics)

	pods, err := c.podLister.Pods(namespace).List(selector)
	if err != nil {
		return 0, 0, 0, time.Time{}, err
	}
	podResources := make(map[string]int64)
	for _, p := range pods {
		var podCpu int64 = 0
		containers := p.Spec.Containers
		for _, cont := range containers {
			cpu := cont.Resources.Requests.Cpu().Value()
			podCpu += cpu
		}
		podResources[p.Name] = podCpu

	}
	return 0, 0, 0, timestamp, nil
}
