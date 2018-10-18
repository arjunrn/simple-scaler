package replicacalculator

import "testing"

func TestMetricsCache_Add(t *testing.T) {
	cache := NewMetricsCache()
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
