package replicacalculator

import (
	"container/list"
	"time"
)

type scalingEvent struct {
	previousReplicas int32
	newReplicas      int32
	timestamp        time.Time
}

type deploymentHistory struct {
	events       *list.List
	lastAccessed time.Time
	eventsLength int
}

// AddEvent adds an event to the deployment history
func (h deploymentHistory) AddEvent(prevReplicas int32, nextReplicas int32) {
	for h.events.Len() >= h.eventsLength {
		front := h.events.Front()
		h.events.Remove(front)
	}
	h.events.PushBack(&scalingEvent{previousReplicas: prevReplicas, newReplicas: nextReplicas, timestamp: time.Now()})
}

func (h deploymentHistory) canScale(cooldown int32, comparator func(int32, int32) bool) bool {

	if cooldown > int32(h.events.Len()) {
		return false
	}
	for i, e := 0, h.events.Back(); i < int(cooldown); i, e = i+1, e.Prev() {
		event := e.Value.(*scalingEvent)
		if comparator(event.previousReplicas, event.newReplicas) {
			return false
		}
	}
	return true
}

func newDeploymentHistory(size int) deploymentHistory {
	return deploymentHistory{
		events:       list.New(),
		lastAccessed: time.Now(),
		eventsLength: size,
	}
}

// DeploymentCache cache for storing deployment events
type DeploymentCache struct {
	deploymentHistories map[string]deploymentHistory
	length              int
	cacheTTL            time.Duration
}

// NewDeploymentCache creates a new DeploymentCache
func NewDeploymentCache(length int, ttl time.Duration) *DeploymentCache {
	return &DeploymentCache{
		deploymentHistories: make(map[string]deploymentHistory),
		length:              length,
		cacheTTL:            ttl,
	}
}

// Gc Cleans up deployment info for outdated Deployments
func (c *DeploymentCache) Gc() {
	oldestAllowed := time.Now().Add(-time.Duration(c.cacheTTL))
	deleteCandidates := make([]string, 0)
	for k, v := range c.deploymentHistories {
		v.lastAccessed.Before(oldestAllowed)
		{
			deleteCandidates = append(deleteCandidates, k)
		}
	}
	for _, k := range deleteCandidates {
		delete(c.deploymentHistories, k)
	}
}

// AddEvent adds an event for a particular deployment
func (c *DeploymentCache) AddEvent(name string, prevReplicas, nextReplicas int32) {
	if _, ok := c.deploymentHistories[name]; !ok {
		c.deploymentHistories[name] = newDeploymentHistory(c.length)
	}
	c.deploymentHistories[name].AddEvent(prevReplicas, nextReplicas)
}

// CanScaleUp checks if the deployment can be scaled up based on scaling information
func (c *DeploymentCache) CanScaleUp(name string, cooldown int32) bool {
	if int(cooldown) > c.length {
		return false
	}
	return c.canScale(name, cooldown, func(a, b int32) bool { return b > a })
}

// CanScaleDown checks if the deployment can be scaled down based on scaling information
func (c *DeploymentCache) CanScaleDown(name string, cooldown int32) bool {
	if int(cooldown) > c.length {
		return false
	}
	return c.canScale(name, cooldown, func(a, b int32) bool { return a > b })
}

func (c *DeploymentCache) canScale(name string, cooldown int32, comparator func(int32, int32) bool) bool {
	history := c.deploymentHistories[name]
	return history.canScale(cooldown, comparator)
}
