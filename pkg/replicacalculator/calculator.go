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
	podLister       corelisters.PodLister
	deploymentCache *DeploymentCache
	metricsGetter   MetricsGetter
}

// NewReplicaCalculator Creates a  new replica calculator
func NewReplicaCalculator(client metrics.MetricsClient, lister corelisters.PodLister, deploymentCache *DeploymentCache, metricsGetter MetricsGetter) *ReplicaCalculator {
	return &ReplicaCalculator{
		podLister:       lister,
		deploymentCache: deploymentCache,
		metricsGetter:   metricsGetter,
	}
}

// GetResourceReplicas get number of replicas for the deployment
func (c *ReplicaCalculator) GetResourceReplicas(currentReplicas int32, downThreshold int32, upThreshold int32, resource v1.ResourceName,
	scaler *v1alpha1.Scaler, selector labels.Selector) (int32, time.Time, error) {
	pods, err := c.podLister.Pods(scaler.Namespace).List(selector)
	if err != nil {
		return -1, time.Time{}, err
	}

	podNames := make([]string, len(pods))
	for i, p := range pods {
		podNames[i] = p.Name
	}

	glog.Infof("%v", podNames)

	metrics, err := c.metricsGetter.GetPodMetrics(scaler.Namespace, podNames)
	if err != nil {
		return -1, time.Time{}, err
	}

	glog.Infof("podmetrics: %v", metrics)

	scaleUp, scaleDown := c.shouldScale(podNames, metrics, int(upThreshold), int(downThreshold))

	if scaleUp && scaleDown {
		scaleDown = false
	}
	proposedReplicas := currentReplicas

	if scaleUp {
		proposedReplicas += 1
	}
	if scaleDown {
		proposedReplicas -= 1
	}

	return proposedReplicas, time.Now(), nil
}

func (c *ReplicaCalculator) shouldScale(podNames []string, podMetrics map[string][]int, scaleUpThreshold, scaleDownThreshold int) (bool, bool) {
	scaleUp := false
	scaleDown := false
	for _, p := range podNames {
		var (
			pMetrics []int
			ok       bool
		)
		if pMetrics, ok = podMetrics[p]; !ok {
			continue
		}

		if len(pMetrics) < 5 {
			continue
		}
		pScaleUp := true
		for _, p := range pMetrics {
			if p < scaleUpThreshold {
				pScaleUp = false
				break
			}
		}
		pScaleDown := true
		for _, p := range pMetrics {
			if p > scaleDownThreshold {
				pScaleDown = false
				break
			}
		}
		scaleDown = pScaleDown
		scaleUp = pScaleUp
	}

	return scaleUp, scaleDown
}
