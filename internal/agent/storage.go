package agent

import (
	"errors"
	"log"
	"math/rand"
	"net/http"
	"reflect"
	"runtime"
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

func (m *Metrics) updateGaugeByRuntimeValue(name string) {
	m.Lock()
	defer m.Unlock()
	mem := &runtime.MemStats{}
	runtime.ReadMemStats(mem)
	m.Gauges[name] = reflect.Indirect(reflect.ValueOf(mem)).FieldByName(name).Convert(reflect.TypeOf(m.Gauges[name])).Float()
}

func (m *Metrics) updateGaugeByRandomValue(name string) {
	m.Lock()
	defer m.Unlock()
	m.Gauges[name] = 100 * rand.Float64()
}

func (m *Metrics) updateCounter(name string) {
	m.Lock()
	defer m.Unlock()
	m.Counters[name] += 1
}

func (m *Metrics) resetCounter(name string) {
	m.Lock()
	defer m.Unlock()
	m.Counters[name] = 0
}

func (m *Metrics) sendGauge(srvAddr, contentType, metricName string) error {
	val, err := m.getGauge(metricName)
	if err != nil {
		log.Println(err.Error())
		return err
	}
	if err = sendPostReqest(srvAddr+"/update/gauge/"+metricName+"/"+strconv.FormatFloat(val, 'f', 3, 64), contentType); err != nil {
		log.Println(err.Error())
		return err
	}
	return nil
}

func (m *Metrics) sendCounter(srvAddr, contentType, metricName string) error {
	val, err := m.getCounter(metricName)
	if err != nil {
		log.Println(err.Error())
		return err
	}
	if err = sendPostReqest(srvAddr+"/update/counter/"+metricName+"/"+strconv.FormatInt(val, 10), contentType); err != nil {
		log.Println(err.Error())
		return err
	}
	storage.resetCounter(metricName)
	return nil
}

func sendPostReqest(addr, cont string) error {
	res, err := http.Post(addr, cont, nil)
	if err != nil {
		log.Println(err.Error())
		return err
	}
	defer res.Body.Close()
	log.Println(res.Status, res.Request.URL)
	return nil
}
