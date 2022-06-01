package server

import (
	"fmt"
	"strconv"

	"sync"
)

var Metrics = struct {
	sync.RWMutex
	Gauges   map[string]float64
	Counters map[string]int64
}{
	Gauges:   map[string]float64{},
	Counters: map[string]int64{},
}

func updGauge(name string, val float64) {
	Metrics.Lock()
	defer Metrics.Unlock()
	Metrics.Gauges[name] = val
}

func updCounter(name string, val int64) {
	Metrics.Lock()
	defer Metrics.Unlock()
	Metrics.Counters[name] += val
}

func getGauges() []string {
	arr := []string{}
	Metrics.RLock()
	defer Metrics.RUnlock()
	for k, v := range Metrics.Gauges {
		arr = append(arr, k+": "+strconv.FormatFloat(v, 'f', 3, 64))
	}
	return arr
}

func getCounters() []string {
	arr := []string{}
	Metrics.RLock()
	defer Metrics.RUnlock()
	for k, v := range Metrics.Counters {
		arr = append(arr, k+": "+strconv.FormatInt(v, 10))
	}
	return arr
}

func getGauge(name string) (float64, error) {
	Metrics.RLock()
	defer Metrics.RUnlock()
	if val, ok := Metrics.Gauges[name]; ok {
		return val, nil
	}
	return 0, fmt.Errorf("cannot get: no such gauge <%v>", name)
}

func getCounter(name string) (int64, error) {
	Metrics.RLock()
	defer Metrics.RUnlock()
	if val, ok := Metrics.Counters[name]; ok {
		return val, nil
	}
	return 0, fmt.Errorf("cannot get: no such counter <%v>", name)
}
