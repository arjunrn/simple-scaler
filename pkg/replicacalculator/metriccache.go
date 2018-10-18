package replicacalculator

import (
	"container/list"
	"time"
)

const (
	bufferMaxSize = 15
)

type MetricsCache struct {
	cache map[string]*ValuesHolder
}

type ValuesHolder struct {
	lastAccessed time.Time
	buffer       *list.List
}

func NewValuesHolder() *ValuesHolder {
	return &ValuesHolder{
		lastAccessed: time.Now(),
		buffer:       list.New(),
	}
}

func (v *ValuesHolder) Add(value int) {
	for v.buffer.Len() >= bufferMaxSize {
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

func NewMetricsCache() *MetricsCache {
	return &MetricsCache{
		cache: make(map[string]*ValuesHolder),
	}
}

func (c *MetricsCache) Add(name string, v int) {
	if _, ok := c.cache[name]; !ok {
		c.cache[name] = NewValuesHolder()
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
	fifteenMsAgo := time.Now().Add(time.Duration(time.Minute * 15))
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
