package replicacalculator

import (
	"container/list"
	"time"
)

const (
	bufferMaxSize = 15
)

// ValueHolder holders metric datapoints with last access time
type ValuesHolder struct {
	lastAccessed time.Time
	buffer       *list.List
	size         int
}

// NewValuesHolder creates a new ValuesHolder
func NewValuesHolder(size int) *ValuesHolder {
	return &ValuesHolder{
		lastAccessed: time.Now(),
		buffer:       list.New(),
		size:         size,
	}
}

// Add adds a datapoint for pod metrics
func (v *ValuesHolder) Add(value int) {
	for v.buffer.Len() >= v.size {
		head := v.buffer.Front()
		v.buffer.Remove(head)
	}
	v.buffer.PushBack(value)
}

func (v *ValuesHolder) Get() []int {
	values := make([]int, v.buffer.Len())
	for i, e := 0, v.buffer.Front(); e != nil; i, e = i+1, e.Next() {
		values[i] = e.Value.(int)
	}
	return values
}

func (v *ValuesHolder) OlderThan(oldTimestamp time.Time) bool {
	return v.lastAccessed.Before(oldTimestamp)
}

// MetricsCache cache for storing pod metrics
type MetricsCache struct {
	cache map[string]*ValuesHolder
	ttl   time.Duration
	size  int
}

func NewMetricsCache(size int, ttl time.Duration) *MetricsCache {
	return &MetricsCache{
		size:  size,
		cache: make(map[string]*ValuesHolder),
		ttl:   ttl,
	}
}

func (c *MetricsCache) Add(name string, v int) {
	if _, ok := c.cache[name]; !ok {
		c.cache[name] = NewValuesHolder(c.size)
	}
	c.cache[name].Add(v)
}

func (c *MetricsCache) Get(name string) []int {
	if _, ok := c.cache[name]; !ok {
		return nil
	}
	return c.cache[name].Get()
}

func (c *MetricsCache) Gc() {
	fifteenMsAgo := time.Now().Add(-c.ttl)
	oldKeys := make([]string, 0)
	for k, cache := range c.cache {
		if cache.OlderThan(fifteenMsAgo) {
			oldKeys = append(oldKeys, k)
		}
	}
	for _, k := range oldKeys {
		delete(c.cache, k)
	}
}

func (c *MetricsCache) IsAboveThreshold(name string, threshold int32, cycles int) bool {
	return c.threshold(name, int(threshold), cycles, func(a, b int) bool { return a < b })
}

func (c *MetricsCache) IsBelowThreshold(name string, threshold int32, cycles int) bool {
	return c.threshold(name, int(threshold), cycles, func(a, b int) bool { return a > b })
}

func (c *MetricsCache) threshold(name string, threshold int, cycles int, comparator func(int, int) bool) bool {
	podCache := c.Get(name)
	if podCache == nil {
		return false
	}
	if len(podCache) < cycles {
		return false
	}
	length := len(podCache)
	for i := 0; i < length; i++ {
		if comparator(podCache[i], threshold) {
			return false
		}
	}
	return true
}
