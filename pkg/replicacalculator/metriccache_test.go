package replicacalculator

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestMetricsCache_Add(t *testing.T) {
	cache := NewMetricsCache(15, time.Duration(time.Minute))
	for i := 0; i < 20; i++ {
		cache.Add("test", i)
	}
	value := cache.Get("test")
	for i, v := range value {
		if i+5 != v {
			t.Errorf("mismatching values %d %d", i, v)
		}
	}
	t.Logf("Passed test")
}

func TestMetricsCache_Gc(t *testing.T) {
	const podName = "test-pod"
	cacheSize := 20
	cache := NewMetricsCache(cacheSize, time.Duration(time.Second))
	for i := 0; i < 100; i++ {
		cache.Add(podName, 10)
	}
	bufferLen := cache.cache[podName].buffer.Len()
	assert.Equal(t, cacheSize, bufferLen, "The number of datapoints do not match %d %d", cacheSize, bufferLen)
	time.Sleep(time.Second)
	cache.Gc()
	_, ok := cache.cache[podName]
	assert.False(t, ok, "The entry is still present after garbage collection")
}
