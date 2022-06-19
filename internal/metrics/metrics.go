package metrics

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
	TextPlainCT = "text/plain"
	JsonCT      = "application/json"
	HttpStr     = "http://"
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

func (m *Metrics) UploadStorage(path string) error {
	log.Println("UPLOAD: " + path)
	return nil
}

func (m *Metrics) DownloadStorage(path string) error {
	log.Println("DOWNLOAD: " + path)
	return nil
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

func (m *Metrics) getMetricJSON(mName, mType string) ([]byte, error) {
	var val float64
	var delta int64
	switch mType {
	case Gauge:
		tmp, err := m.GetGauge(mName)
		if err != nil {
			log.Println(err.Error())
			return []byte{}, err
		}
		val = tmp
	case Counter:
		tmp, err := m.GetCounter(mName)
		if err != nil {
			log.Println(err.Error())
			return []byte{}, err
		}
		delta = tmp
	default:
		err := errors.New("unknown metric type <" + mType + ">")
		log.Println(err)
		return []byte{}, err
	}
	mj, err := json.Marshal(MetricJSON{
		ID:    mName,
		MType: mType,
		Value: &val,
		Delta: &delta,
	})
	if err != nil {
		log.Println(err.Error())
		return []byte{}, err
	}
	return mj, nil
}

func (m *Metrics) GetGauge(name string) (float64, error) {
	m.RLock()
	defer m.RUnlock()
	if val, ok := m.Gauges[name]; ok {
		return val, nil
	}
	err := errors.New("cannot get: no such gauge <" + name + ">")
	log.Println(err.Error())
	return 0, err
}

func (m *Metrics) GetCounter(name string) (int64, error) {
	m.RLock()
	defer m.RUnlock()
	if val, ok := m.Counters[name]; ok {
		return val, nil
	}
	err := errors.New("cannot get: no such counter <" + name + ">")
	log.Println(err.Error())
	return 0, err
}

func (m *Metrics) GetGauges() []string {
	arr := []string{}
	m.RLock()
	defer m.RUnlock()
	for k, v := range m.Gauges {
		arr = append(arr, k+": "+strconv.FormatFloat(v, 'f', 3, 64))
	}
	return arr
}

func (m *Metrics) GetCounters() []string {
	arr := []string{}
	m.RLock()
	defer m.RUnlock()
	for k, v := range m.Counters {
		arr = append(arr, k+": "+strconv.FormatInt(v, 10))
	}
	return arr
}

func (m *Metrics) UpdateMetricFromServer(srvAddr, mName, mType string) error {
	body, err := getEmptyMetricJSON(mName, mType)
	if err != nil {
		log.Println(err.Error())
		return err
	}
	res, err := http.Post(HttpStr+srvAddr+"/value/", JsonCT, bytes.NewBuffer(body))
	if err != nil {
		log.Println(err.Error())
		return err
	}
	defer res.Body.Close()
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
	fmt.Println(mj)
	switch mj.MType {
	case Gauge:
		m.UpdateGaugeByValue(mj.ID, *mj.Value)
	case Counter:
		m.UpdateCounterByValue(mj.ID, *mj.Delta)
	default:
		err := errors.New("cannot update: unknown metric type <" + mj.ID + ">")
		log.Println(err.Error())
		return err
	}
	return nil
}

func (m *Metrics) UpdateGaugeByValue(name string, val float64) {
	m.Lock()
	defer m.Unlock()
	m.Gauges[name] = val
}

func (m *Metrics) UpdateGaugeByRuntimeValue(name string) {
	m.Lock()
	defer m.Unlock()
	mem := &runtime.MemStats{}
	runtime.ReadMemStats(mem)
	m.Gauges[name] = reflect.Indirect(reflect.ValueOf(mem)).FieldByName(name).Convert(reflect.TypeOf(m.Gauges[name])).Float()
}

func (m *Metrics) UpdateGaugeByRandomValue(name string) {
	m.Lock()
	defer m.Unlock()
	m.Gauges[name] = 100 * rand.Float64()
}

func (m *Metrics) UpdateCounterByValue(name string, val int64) {
	m.Lock()
	defer m.Unlock()
	m.Counters[name] = val
}

func (m *Metrics) IncreaseCounter(name string, val int64) {
	m.Lock()
	defer m.Unlock()
	m.Counters[name] += val
}

func (m *Metrics) ResetCounter(name string) {
	m.Lock()
	defer m.Unlock()
	m.Counters[name] = 0
}

func (m *Metrics) SendMetric(srvAddr, contentType, mName, mType string) error {
	var url string
	var body []byte
	switch contentType {
	case TextPlainCT:
		switch mType {
		case Gauge:
			val, err := m.GetGauge(mName)
			if err != nil {
				log.Println(err.Error())
				return err
			}
			url = srvAddr + "/update/gauge/" + mName + "/" + strconv.FormatFloat(val, 'f', 3, 64)
			body = nil
		case Counter:
			val, err := m.GetCounter(mName)
			if err != nil {
				log.Println(err.Error())
				return err
			}
			url = srvAddr + "/update/counter/" + mName + "/" + strconv.FormatInt(val, 10)
			body = nil
		default:
			err := errors.New("unknown metric type <" + mType + ">")
			log.Println(err)
			return err
		}
	case JsonCT:
		tmp, err := m.getMetricJSON(mName, mType)
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
	res, err := http.Post(HttpStr+url, contentType, bytes.NewBuffer(body))
	if err != nil {
		log.Println(err.Error())
		return err
	}
	defer res.Body.Close()
	if mType == Counter {
		m.ResetCounter(mName)
	}
	log.Println(res.Status, res.Request.URL)
	return nil
}
