package metrics

import (
	"bufio"
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"reflect"
	"runtime"
	"strconv"
	"sync"
)

const (
	Gauge       = "gauge"
	Counter     = "counter"
	TextPlainCT = "text/plain"
	JSONCT      = "application/json"
	HTTPStr     = "http://"
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
	Hash  string   `json:"hash,omitempty"`
}

func (m *Metrics) UploadStorage(path string) error {
	m.Lock()
	defer m.Unlock()
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0777)
	if err != nil {
		log.Println(err.Error())
		return err
	}
	defer file.Close()
	for key := range m.Gauges {
		mj, err := m.getMetricJSON(key, Gauge)
		if err != nil {
			log.Println(err.Error())
			return err
		}
		mj = append(mj, '\n')
		_, err = file.Write(mj)
		if err != nil {
			log.Println(err.Error())
			return err
		}
	}
	for key := range m.Counters {
		mj, err := m.getMetricJSON(key, Counter)
		if err != nil {
			log.Println(err.Error())
			return err
		}
		mj = append(mj, '\n')
		_, err = file.Write(mj)
		if err != nil {
			log.Println(err.Error())
			return err
		}
	}
	log.Println("UPLOADED TO: " + path)
	return nil
}

func (m *Metrics) DownloadStorage(path string) error {
	file, err := os.Open(path)
	if err != nil {
		log.Println(err.Error())
		return err
	}
	defer file.Close()
	b := bufio.NewScanner(file)
	for b.Scan() {
		m.UpdateMetricFromJSON(b.Bytes())
	}
	log.Println("DOWNLOADED FROM: " + path)
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
	switch mType {
	case Gauge:
		val, err := m.getGauge(mName)
		if err != nil {
			log.Println(err.Error())
			return []byte{}, err
		}
		mj, err := json.Marshal(MetricJSON{
			ID:    mName,
			MType: mType,
			Value: &val,
		})
		if err != nil {
			log.Println(err.Error())
			return []byte{}, err
		}
		return mj, nil
	case Counter:
		val, err := m.getCounter(mName)
		if err != nil {
			log.Println(err.Error())
			return []byte{}, err
		}
		mj, err := json.Marshal(MetricJSON{
			ID:    mName,
			MType: mType,
			Delta: &val,
		})
		if err != nil {
			log.Println(err.Error())
			return []byte{}, err
		}
		return mj, nil
	default:
		err := errors.New("unknown metric type <" + mType + ">")
		log.Println(err)
		return []byte{}, err
	}
}

func (m *Metrics) GetGauge(name string) (float64, error) {
	m.Lock()
	defer m.Unlock()
	return m.getGauge(name)
}

func (m *Metrics) getGauge(name string) (float64, error) {
	if val, ok := m.Gauges[name]; ok {
		return val, nil
	}
	err := errors.New("cannot get: no such gauge <" + name + ">")
	log.Println(err.Error())
	return 0, err
}

func (m *Metrics) GetCounter(name string) (int64, error) {
	m.Lock()
	defer m.Unlock()
	return m.getCounter(name)
}

func (m *Metrics) getCounter(name string) (int64, error) {
	if val, ok := m.Counters[name]; ok {
		return val, nil
	}
	err := errors.New("cannot get: no such counter <" + name + ">")
	log.Println(err.Error())
	return 0, err
}

func (m *Metrics) GetGauges() []string {
	arr := []string{}
	m.Lock()
	defer m.Unlock()
	for k, v := range m.Gauges {
		arr = append(arr, k+": "+strconv.FormatFloat(v, 'f', 3, 64))
	}
	return arr
}

func (m *Metrics) GetCounters() []string {
	arr := []string{}
	m.Lock()
	defer m.Unlock()
	for k, v := range m.Counters {
		arr = append(arr, k+": "+strconv.FormatInt(v, 10))
	}
	return arr
}

func (m *Metrics) UpdateMetricFromJSON(content []byte) error {
	mj := MetricJSON{}
	if err := json.Unmarshal(content, &mj); err != nil {
		log.Println(err.Error())
		return err
	}
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

func (m *Metrics) UpdateMetricFromServer(srvAddr, mName, mType string) error {
	body, err := getEmptyMetricJSON(mName, mType)
	if err != nil {
		log.Println(err.Error())
		return err
	}
	res, err := http.Post(HTTPStr+srvAddr+"/value/", JSONCT, bytes.NewBuffer(body))
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
	err = m.UpdateMetricFromJSON(content)
	if err != nil {
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
	case JSONCT:
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
	res, err := customPostRequest(HTTPStr+url, contentType, "", bytes.NewBuffer(body))
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

func customPostRequest(url, contentType, hash string, body io.Reader) (resp *http.Response, err error) {
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}
	if hash != "" {
		req.Header.Set("Hash", hash)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	return http.DefaultClient.Do(req)
}

func hash(source, key string) ([]byte, error) {
	h := hmac.New(sha256.New, []byte(key))
	_, err := h.Write([]byte(source))
	if err != nil {
		log.Println(err.Error())
		return nil, err
	}
	return h.Sum(nil), nil
}
