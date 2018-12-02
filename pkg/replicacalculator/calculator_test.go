package replicacalculator

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewReplicaCalculator(t *testing.T) {
	testCases := []struct {
		podMetrics                                        map[string][]int
		podNames                                          []string
		scaleUp, scaleDown                                bool
		scaleUpThreshold, scaleDownThreshold, evaluations int
	}{
		{
			podMetrics:         map[string][]int{"abc": {30, 31, 32, 33, 34}, "def": {0, 0, 0, 0, 0}},
			podNames:           []string{"abc", "def"},
			scaleUp:            true,
			scaleDown:          true,
			scaleUpThreshold:   30,
			scaleDownThreshold: 10,
			evaluations:        5,
		},
	}

	calculator := ReplicaCalculator{}
	for _, c := range testCases {
		scaleUp, scaleDown := calculator.shouldScale(c.podNames, c.podMetrics, c.scaleUpThreshold, c.scaleDownThreshold, c.evaluations)
		assert.Equal(t, c.scaleUp, scaleUp, "scale up should be %v instead is %v", c.scaleUp, scaleUp)
		assert.True(t, c.scaleDown, scaleDown, "scale down should be %v instead is %v", c.scaleDown, scaleDown)
	}
}
