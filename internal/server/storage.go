package server

import (
	"errors"
	"log"
	"strconv"
	"sync"
)

const (
	Gauge       = "gauge"
	Counter     = "counter"
	textPlainCT = "text/plain"
	jsonCT      = "application/json"
)

type Metrics struct {
	sync.RWMutex
	Gauges   map[string]float64
	Counters map[string]int64
}

type MetricJSON struct {
	ID    string   `json:"id"`
	MType string   `json:"type"`
	Delta *int64   `json:"delta,omitempty"`
	Value *float64 `json:"value,omitempty"`
}

func (m *Metrics) updateGauge(name string, val float64) {
	m.Lock()
	defer m.Unlock()
	m.Gauges[name] = val
}

func (m *Metrics) updateCounter(name string, val int64) {
	m.Lock()
	defer m.Unlock()
	m.Counters[name] += val
}

func (m *Metrics) getGauges() []string {
	arr := []string{}
	m.RLock()
	defer m.RUnlock()
	for k, v := range m.Gauges {
		arr = append(arr, k+": "+strconv.FormatFloat(v, 'f', 3, 64))
	}
	return arr
}

func (m *Metrics) getCounters() []string {
	arr := []string{}
	m.RLock()
	defer m.RUnlock()
	for k, v := range m.Counters {
		arr = append(arr, k+": "+strconv.FormatInt(v, 10))
	}
	return arr
}

func (m *Metrics) getGauge(name string) (float64, error) {
	m.RLock()
	defer m.RUnlock()
	if val, ok := m.Gauges[name]; ok {
		return val, nil
	}
	err := errors.New("cannot get: no such gauge <" + name + ">")
	log.Println(err.Error())
	return 0, err
}

func (m *Metrics) getCounter(name string) (int64, error) {
	m.RLock()
	defer m.RUnlock()
	if val, ok := m.Counters[name]; ok {
		return val, nil
	}
	err := errors.New("cannot get: no such counter <" + name + ">")
	log.Println(err.Error())
	return 0, err
}
