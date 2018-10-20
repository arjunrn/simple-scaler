package replicacalculator

import (
	"container/list"
	"fmt"
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

type DeploymentCache struct {
	deploymentHistories map[string]deploymentHistory
	length              int
	cacheTTL            time.Duration
}

func NewDeploymentCache(length int, ttl time.Duration) *DeploymentCache {
	return &DeploymentCache{
		deploymentHistories: make(map[string]deploymentHistory),
		length:              length,
		cacheTTL:            ttl,
	}
}

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

func (c *DeploymentCache) AddEvent(name string, prevReplicas, nextReplicas int32) {
	if _, ok := c.deploymentHistories[name]; !ok {
		c.deploymentHistories[name] = newDeploymentHistory(c.length)
	}
	c.deploymentHistories[name].AddEvent(prevReplicas, nextReplicas)
}

func (c *DeploymentCache) CanScaleUp(name string, cooldown int32) (bool, error) {
	if int(cooldown) > c.length {
		return false, fmt.Errorf("cooldown %d is longer than history %d", cooldown, c.length)
	}
	return c.canScale(name, cooldown, func(a, b int32) bool { return b > a }), nil
}
func (c *DeploymentCache) CanScaleDown(name string, cooldown int32) (bool, error) {
	if int(cooldown) > c.length {
		return false, fmt.Errorf("cooldown %d is longer than history %d", cooldown, c.length)
	}
	return c.canScale(name, cooldown, func(a, b int32) bool { return a > b }), nil
}

func (c *DeploymentCache) canScale(name string, cooldown int32, comparator func(int32, int32) bool) bool {
	history := c.deploymentHistories[name]
	return history.canScale(cooldown, comparator)
}
