package replicacalculator

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestNewDeploymentCache(t *testing.T) {
	ttl := time.Duration(time.Minute * 15)
	cache := NewDeploymentCache(5, ttl)
	deploymentName := "test-deployment"
	for i := 0; i < 10; i++ {
		cache.AddEvent(deploymentName, 5, 5)
	}
	_, err := cache.CanScaleUp(deploymentName, 10)
	assert.Errorf(t, err, "no error when  cooldown larger than cache size")

	ttl = time.Duration(1 * time.Second)
	cache = NewDeploymentCache(10, ttl)
	for i := 0; i < 100; i++ {
		cache.AddEvent(deploymentName, 5, 5)
	}
	eventsLength := cache.deploymentHistories[deploymentName].events.Len()
	assert.Equal(t, eventsLength, 10, "cache size %d is larger than specified %d", eventsLength, 10)
	time.Sleep(ttl)
	cache.Gc()
	_, ok := cache.deploymentHistories[deploymentName]
	assert.False(t, ok, "events cache is present even after gc")
}

func TestDeploymentCache_CanScaleUp(t *testing.T) {
	ttl := time.Duration(time.Minute * 15)
	cache := NewDeploymentCache(5, ttl)
	deploymentName := "test-deployment"
	cache.AddEvent(deploymentName, 1, 1)
	cache.AddEvent(deploymentName, 1, 1)
	cache.AddEvent(deploymentName, 1, 3)
	allowed, err := cache.CanScaleUp(deploymentName, 3)
	assert.NoError(t, err, "error with cache size")
	assert.False(t, allowed, "allowed to scale up when when scale up happened in last 3 cycles")

	cache = NewDeploymentCache(5, ttl)
	cache.AddEvent(deploymentName, 1, 3)
	cache.AddEvent(deploymentName, 1, 1)
	cache.AddEvent(deploymentName, 1, 1)
	cache.AddEvent(deploymentName, 1, 1)
	allowed, err = cache.CanScaleUp(deploymentName, 3)
	assert.NoError(t, err, "error with cache size")
	assert.True(t, allowed, "not allowed to scale up even when scale up not happened in last 3 cycles")
}

func TestDeploymentCache_CanScaleDown(t *testing.T) {
	ttl := time.Duration(time.Minute * 15)
	cache := NewDeploymentCache(5, ttl)
	deploymentName := "test-deployment"
	cache.AddEvent(deploymentName, 3, 3)
	cache.AddEvent(deploymentName, 3, 3)
	cache.AddEvent(deploymentName, 3, 2)
	allowed, err := cache.CanScaleDown(deploymentName, 3)
	assert.NoError(t, err, "error with cache size")
	assert.False(t, allowed, "not allowed to scale down even when no scale down events in 3 cycles")

	cache = NewDeploymentCache(5, ttl)
	cache.AddEvent(deploymentName, 3, 2)
	cache.AddEvent(deploymentName, 3, 3)
	cache.AddEvent(deploymentName, 3, 3)
	cache.AddEvent(deploymentName, 3, 3)
	allowed, err = cache.CanScaleDown(deploymentName, 3)
	assert.NoError(t, err, "error with cache size")
	assert.True(t, allowed, "not allowed to scale down even when scale down not happened in last 3 cycles")
}
