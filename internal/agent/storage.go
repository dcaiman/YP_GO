package agent

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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

type MetricJSON struct {
	ID    string   `json:"id"`
	MType string   `json:"type"`
	Delta *int64   `json:"delta,omitempty"`
	Value *float64 `json:"value,omitempty"`
}

func getEmptyMetricJSON(mName, mType string) ([]byte, error) {
	j, err := json.Marshal(MetricJSON{
		ID:    mName,
		MType: mType,
	})
	if err != nil {
		log.Println(err.Error())
		return []byte{}, err
	}
	return j, nil
}

func (m *Metrics) getGaugeJSON(name string) ([]byte, error) {
	val, err := m.getGauge(name)
	if err != nil {
		log.Println(err.Error())
		return []byte{}, err
	}
	mj, err := json.Marshal(MetricJSON{
		ID:    name,
		MType: Gauge,
		Value: &val,
	})
	if err != nil {
		log.Println(err.Error())
		return []byte{}, err
	}
	return mj, nil
}

func (m *Metrics) getCounterJSON(name string) ([]byte, error) {
	val, err := m.getCounter(name)
	if err != nil {
		log.Println(err.Error())
		return []byte{}, err
	}
	mj, err := json.Marshal(MetricJSON{
		ID:    name,
		MType: Counter,
		Delta: &val,
	})
	if err != nil {
		log.Println(err.Error())
		return []byte{}, err
	}
	return mj, nil
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

func (m *Metrics) updateMetricFromServer(srvAddr, mName, mType string) error {
	body, err := getEmptyMetricJSON(mName, mType)
	if err != nil {
		log.Println(err.Error())
		return err
	}
	res, err := http.Post(srvAddr+"/value/", jsonCT, bytes.NewBuffer(body))
	if err != nil {
		log.Println(err.Error())
		return err
	}
	defer res.Body.Close()
	fmt.Println(res)
	content, err := io.ReadAll(res.Body)
	if err != nil {
		log.Println(err.Error())
		return err
	}
	mj := MetricJSON{}
	if err := json.Unmarshal(content, &mj); err != nil {
		log.Println(err.Error())
		return err
	}
	switch mj.MType {
	case Gauge:
		storage.updateGaugeByValue(mj.ID, *mj.Value)
	case Counter:
		storage.updateCopunterByValue(mj.ID, *mj.Delta)
		v, _ := m.getCounter(mj.ID)
		fmt.Println(v)
	default:
		err := errors.New("cannot update: no such metric type <" + mj.ID + ">")
		log.Println(err.Error())
		return err
	}
	return nil
}

func (m *Metrics) updateGaugeByValue(name string, val float64) {
	m.Lock()
	defer m.Unlock()
	m.Gauges[name] = val
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

func (m *Metrics) updateCopunterByValue(name string, val int64) {
	m.Lock()
	defer m.Unlock()
	m.Counters[name] = val
}

func (m *Metrics) updateCounter(name string, val int64) {
	m.Lock()
	defer m.Unlock()
	m.Counters[name] += val
}

func (m *Metrics) resetCounter(name string) {
	m.Lock()
	defer m.Unlock()
	m.Counters[name] = 0
}

func (m *Metrics) sendGauge(srvAddr, contentType, metricName string) error {
	var url string
	var body []byte
	switch contentType {
	case textPlainCT:
		val, err := m.getGauge(metricName)
		if err != nil {
			log.Println(err.Error())
			return err
		}
		url = srvAddr + "/update/gauge/" + metricName + "/" + strconv.FormatFloat(val, 'f', 3, 64)
		body = nil
	case jsonCT:
		tmp, err := m.getGaugeJSON(metricName)
		if err != nil {
			log.Println(err.Error())
			return err
		}
		url = srvAddr + "/update/"
		body = tmp
	default:
		err := errors.New("unsupported content type <" + contentType + ">")
		log.Println(err)
		return err
	}
	res, err := http.Post(url, contentType, bytes.NewBuffer(body))
	if err != nil {
		log.Println(err.Error())
		return err
	}
	defer res.Body.Close()
	log.Println(res.Status, res.Request.URL)
	return nil
}

func (m *Metrics) sendCounter(srvAddr, contentType, metricName string) error {
	var url string
	var body []byte
	switch contentType {
	case textPlainCT:
		val, err := m.getCounter(metricName)
		if err != nil {
			log.Println(err.Error())
			return err
		}
		url = srvAddr + "/update/counter/" + metricName + "/" + strconv.FormatInt(val, 10)
		body = nil
	case jsonCT:
		tmp, err := m.getCounterJSON(metricName)
		if err != nil {
			log.Println(err.Error())
			return err
		}
		url = srvAddr + "/update/"
		body = tmp
	default:
		err := errors.New("unsupported content type <" + contentType + ">")
		log.Println(err)
		return err
	}
	res, err := http.Post(url, contentType, bytes.NewBuffer(body))
	if err != nil {
		log.Println(err.Error())
		return err
	}
	defer res.Body.Close()
	log.Println(res.Status, res.Request.URL)
	storage.resetCounter(metricName)
	return nil
}
