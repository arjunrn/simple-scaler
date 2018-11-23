package replicacalculator

import (
	"github.com/arjunrn/dumb-scaler/pkg/apis/scaler/v1alpha1"
	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	corelisters "k8s.io/client-go/listers/core/v1"
)

type ReplicaCalculator struct {
	podLister         corelisters.PodLister
	prometheusMetrics MetricsSource
}

// NewReplicaCalculator Creates a  new replica calculator
func NewReplicaCalculator(lister corelisters.PodLister, prometheusMetrics MetricsSource) *ReplicaCalculator {
	return &ReplicaCalculator{
		podLister:         lister,
		prometheusMetrics: prometheusMetrics,
	}
}

// GetResourceReplicas get number of replicas for the deployment
func (c *ReplicaCalculator) GetResourceReplicas(currentReplicas int32, downThreshold int32, upThreshold int32, resource v1.ResourceName,
	scaler *v1alpha1.Scaler, selector labels.Selector) (int32, error) {
	pods, err := c.podLister.Pods(scaler.Namespace).List(selector)
	if err != nil {
		return -1, err
	}

	podNames := make([]string, len(pods))
	for i, p := range pods {
		podNames[i] = p.Name
	}

	log.Infof("%v", podNames)

	metrics, err := c.prometheusMetrics.GetPodMetrics(scaler.Namespace, podNames)
	if err != nil {
		return -1, err
	}

	log.Infof("pod metrics: %v", metrics)

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

	return proposedReplicas, nil
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
