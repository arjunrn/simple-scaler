package replicacalculator

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewReplicaCalculator(t *testing.T) {
	testCases := []struct {
		name                                              string
		podMetrics                                        map[string][]int
		podNames                                          []string
		scaleUp, scaleDown                                bool
		scaleUpThreshold, scaleDownThreshold, evaluations int
	}{
		{
			name:               "metrics missing for pods and scaling up",
			podMetrics:         map[string][]int{"abc": {30, 31, 32, 33, 34}, "def": {}},
			podNames:           []string{"abc", "def"},
			scaleUp:            true,
			scaleDown:          false,
			scaleUpThreshold:   30,
			scaleDownThreshold: 10,
			evaluations:        5,
		},
		{
			name:               "metrics missing for one of the pods and scale down true",
			podMetrics:         map[string][]int{"abc": {}, "def": {0, 0, 0, 0, 0}},
			podNames:           []string{"abc", "def"},
			scaleUp:            false,
			scaleDown:          true,
			scaleUpThreshold:   30,
			scaleDownThreshold: 10,
			evaluations:        5,
		},
		{
			name:               "both scaling up and scaling down are true",
			podMetrics:         map[string][]int{"abc": {30, 31, 32, 33, 34}, "def": {0, 0, 0, 0, 0}},
			podNames:           []string{"abc", "def"},
			scaleUp:            true,
			scaleDown:          true,
			scaleUpThreshold:   30,
			scaleDownThreshold: 10,
			evaluations:        5,
		},
		{
			name:               "insufficient metrics for evaluations",
			podMetrics:         map[string][]int{"abc": {30, 31, 32, 33, 34}, "def": {0, 0, 0, 0, 0}},
			podNames:           []string{"abc", "def"},
			scaleUp:            false,
			scaleDown:          false,
			scaleUpThreshold:   30,
			scaleDownThreshold: 10,
			evaluations:        6,
		},
		{
			name:               "on the edge",
			podMetrics:         map[string][]int{"abc": {30, 30, 30, 30, 30}, "def": {20, 20, 20, 20, 20}},
			podNames:           []string{"abc", "def"},
			scaleUp:            true,
			scaleDown:          true,
			scaleUpThreshold:   30,
			scaleDownThreshold: 20,
			evaluations:        5,
		},
		{
			name:               "one outlier point",
			podMetrics:         map[string][]int{"abc": {30, 30, 29, 30, 30}, "def": {20, 20, 21, 20, 20}},
			podNames:           []string{"abc", "def"},
			scaleUp:            false,
			scaleDown:          false,
			scaleUpThreshold:   30,
			scaleDownThreshold: 20,
			evaluations:        5,
		},
	}

	calculator := ReplicaCalculator{}
	for _, c := range testCases {
		t.Run(c.name, func(t *testing.T) {
			scaleUp, scaleDown := calculator.shouldScale(c.podNames, c.podMetrics, c.scaleUpThreshold, c.scaleDownThreshold, c.evaluations)
			assert.Equal(t, c.scaleUp, scaleUp, "scale up should be %t instead is %t", c.scaleUp, scaleUp)
			assert.Equal(t, c.scaleDown, scaleDown, "scale down should be %t instead is %t", c.scaleDown, scaleDown)
		})
	}
}
