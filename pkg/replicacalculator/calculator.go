package replicacalculator

import (
	"github.com/arjunrn/dumb-scaler/pkg/apis/scaler/v1alpha1"
	"github.com/golang/glog"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/kubernetes/pkg/controller/podautoscaler/metrics"
	"time"
)

type ReplicaCalculator struct {
	metricsClient   metrics.MetricsClient
	podLister       corelisters.PodLister
	deploymentCache *DeploymentCache
	metricsCache    *MetricsCache
}

// NewReplicaCalculator Creates a  new replica calculator
func NewReplicaCalculator(client metrics.MetricsClient, lister corelisters.PodLister, deploymentCache *DeploymentCache) *ReplicaCalculator {
	return &ReplicaCalculator{
		metricsClient:   client,
		podLister:       lister,
		metricsCache:    NewMetricsCache(15, 15*time.Minute),
		deploymentCache: deploymentCache,
	}
}

// GetResourceReplicas get number of replicas for the deployment
func (c *ReplicaCalculator) GetResourceReplicas(currentReplicas int32, downThreshold int32, upThreshold int32, resource v1.ResourceName,
	scaler *v1alpha1.Scaler, selector labels.Selector) (int32, time.Time, error) {
	metrics, timestamp, err := c.metricsClient.GetResourceMetric(resource, scaler.Namespace, selector)
	if err != nil {
		return 0, time.Now(), err
	}
	glog.Infof("metrics: %v", metrics)

	pods, err := c.podLister.Pods(scaler.Namespace).List(selector)
	if err != nil {
		return 0, timestamp, err
	}
	podResources := make(map[string]int64)
	for _, p := range pods {
		var podCpu int64 = 0
		containers := p.Spec.Containers
		for _, cont := range containers {
			cpu := cont.Resources.Requests.Cpu().MilliValue()
			podCpu += cpu
		}
		podResources[p.Name] = podCpu

	}

	allPods := getResourcePodUtilizations(podResources, metrics)
	glog.Infof("Utilizations for %d pods", len(allPods))

	for k, v := range allPods {
		c.metricsCache.Add(k, v)
	}
	scaleUp, scaleDown := false, false

	for k := range allPods {
		if c.metricsCache.IsAboveThreshold(k, upThreshold, 3) {
			scaleUp = true
			break
		}
	}
	for k := range allPods {
		if c.metricsCache.IsBelowThreshold(k, upThreshold, 3) {
			scaleDown = true
			break
		}
	}

	if scaleUp && scaleDown {
		scaleDown = false
	}

	proposedReplicas := currentReplicas
	if scaleUp && c.deploymentCache.CanScaleUp(scaler.Name, 3) {
		proposedReplicas += 1
		glog.Infof("Scaling Up")
	} else if scaleDown && c.deploymentCache.CanScaleDown(scaler.Name, 3) {
		proposedReplicas -= 1
		glog.Infof("Scaling Down")
	} else {
		glog.Infof("No Scaling Activity")
	}

	return proposedReplicas, timestamp, nil
}

func getResourcePodUtilizations(podResources map[string]int64, podMetricsInfos metrics.PodMetricsInfo) map[string]int {
	allPods := make(map[string]int, len(podResources))
	i := 0
	for k := range podResources {
		allPods[k] = 0
		i++
	}
	for k := range podMetricsInfos {
		allPods[k] = 0
	}

	for k := range allPods {
		resources, ok := podResources[k]
		if !ok {
			allPods[k] = 100
			continue
		} else {
			metric, ok := podMetricsInfos[k]
			if !ok {
				allPods[k] = 100
				continue
			}
			if resources != 0 {
				allPods[k] = int(float64(metric.Value) / float64(resources) * 100)
			} else {
				allPods[k] = 100
			}

		}
	}

	return allPods
}
