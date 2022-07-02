package metrics

import (
	"bufio"
	"errors"
	"log"
	"os"
	"sync"
)

type MetricStorage struct {
	sync.RWMutex
	EncryptingKey string
	Metrics       map[string]Metric
}

func (st *MetricStorage) Init() {
	st.Metrics = map[string]Metric{}
}

func (st *MetricStorage) UploadStorage(path string) error {
	st.Lock()
	defer st.Unlock()
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0777)
	if err != nil {
		return err
	}
	defer file.Close()
	for name := range st.Metrics {
		mj, err := st.Metrics[name].GetJSON()
		if err != nil {
			return err
		}
		mj = append(mj, '\n')
		_, err = file.Write(mj)
		if err != nil {
			return err
		}
	}
	log.Println("UPLOADED TO: " + path)
	return nil
}

func (st *MetricStorage) DownloadStorage(path string) error {
	st.Lock()
	defer st.Unlock()
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	b := bufio.NewScanner(file)
	for b.Scan() {
		st.updateMetricFromJSON(b.Bytes())
	}
	log.Println("DOWNLOADED FROM: " + path)
	return nil
}

func (st *MetricStorage) UpdateMetricFromJSON(content []byte) error {
	st.Lock()
	defer st.Unlock()
	return st.updateMetricFromJSON(content)
}

func (st *MetricStorage) updateMetricFromJSON(content []byte) error {
	m, err := SetFromJSON(&Metric{}, content)
	if err != nil {
		return err
	}
	st.Metrics[m.ID] = m
	return nil
}

func (st *MetricStorage) UpdateMetricFromStruct(m Metric) {
	st.Lock()
	defer st.Unlock()
	st.updateMetricFromStruct(m)
}

func (st *MetricStorage) updateMetricFromStruct(m Metric) {
	st.Metrics[m.ID] = m
}

func (st *MetricStorage) MetricExists(mName, mType string) bool {
	st.Lock()
	defer st.Unlock()
	if m, ok := st.Metrics[mName]; ok {
		if m.MType == mType {
			return true
		}
	}
	return false
}

func (st *MetricStorage) NewMetric(mName, mType string, value *float64, delta *int64) error {
	st.Lock()
	defer st.Unlock()
	return st.newMetric(mName, mType, value, delta)
}

func (st *MetricStorage) newMetric(mName, mType string, value *float64, delta *int64) error {
	if _, ok := st.Metrics[mName]; ok {
		err := errors.New("cannot create: metric <" + mName + "> already exists")
		return err
	}
	m := &Metric{
		ID:    mName,
		MType: mType,
		Value: value,
		Delta: delta,
	}
	m.UpdateHash(st.EncryptingKey)
	st.updateMetricFromStruct(*m)
	return nil
}

func (st *MetricStorage) GetMetric(name string) (Metric, error) {
	st.Lock()
	defer st.Unlock()
	if m, ok := st.Metrics[name]; ok {
		return m, nil
	}
	err := errors.New("cannot get: metric <" + name + "> doesn't exist")
	return Metric{}, err
}

func (st *MetricStorage) UpdateValue(name string, val float64) error {
	st.Lock()
	defer st.Unlock()
	return st.updateValue(name, val)
}

func (st *MetricStorage) updateValue(name string, val float64) error {
	if m, ok := st.Metrics[name]; ok {
		m.Value = &val
		err := m.UpdateHash(st.EncryptingKey)
		if err != nil {
			return err
		}
		st.Metrics[name] = m
		return nil
	}
	err := errors.New("cannot update Value: metric <" + name + "> doesn't exist")
	return err
}

func (st *MetricStorage) UpdateDelta(name string, val int64) error {
	st.Lock()
	defer st.Unlock()
	return st.updateDelta(name, val)
}

func (st *MetricStorage) updateDelta(name string, val int64) error {
	if m, ok := st.Metrics[name]; ok {
		m.Delta = &val
		err := m.UpdateHash(st.EncryptingKey)
		if err != nil {
			return err
		}
		st.Metrics[name] = m
		return nil
	}
	err := errors.New("cannot update Delta: metric <" + name + "> doesn't exist")
	return err
}

func (st *MetricStorage) AddDelta(name string, val int64) error {
	st.Lock()
	defer st.Unlock()
	currentVal := st.Metrics[name].Delta
	if currentVal == nil {
		return st.updateDelta(name, val)
	}
	return st.updateDelta(name, *currentVal+val)
}

func (st *MetricStorage) IncreaseDelta(name string) error {
	st.Lock()
	defer st.Unlock()
	val := st.Metrics[name].Delta
	if val == nil {
		return st.updateDelta(name, 1)
	}
	return st.updateDelta(name, *val+1)
}

func (st *MetricStorage) ResetDelta(name string) error {
	st.Lock()
	defer st.Unlock()
	return st.updateDelta(name, 0)
}
