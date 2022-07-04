package internalstorage

import (
	"bufio"
	"context"
	"errors"
	"log"
	"os"
	"sync"
	"text/template"

	"github.com/dcaiman/YP_GO/internal/metric"
)

const (
	templateHandlerGetAll = "METRICS LIST: <p>{{range .Metrics}}{{.ID}}: {{.Value}}{{.Delta}} ({{.MType}})</p>{{end}}"
)

type MetricStorage struct {
	sync.RWMutex
	Path    string
	Metrics map[string]metric.Metric
}

func (st *MetricStorage) Init(path string) {
	st.Lock()
	defer st.Unlock()
	st.Path = path
	st.Metrics = map[string]metric.Metric{}
}

func (st *MetricStorage) AccessCheck(ctx context.Context) error {
	if st.Metrics == nil {
		err := errors.New("storage map is not initialized")
		return err
	}
	return nil
}

func (st *MetricStorage) GetHTML() (*template.Template, error) {
	t, err := template.New("").Parse(templateHandlerGetAll)
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (st *MetricStorage) UploadStorage() error {
	st.Lock()
	defer st.Unlock()
	file, err := os.OpenFile(st.Path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0777)
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
	log.Println("UPLOADED TO: " + st.Path)
	return nil
}

func (st *MetricStorage) DownloadStorage() error {
	st.Lock()
	defer st.Unlock()
	file, err := os.Open(st.Path)
	if err != nil {
		return err
	}
	defer file.Close()
	b := bufio.NewScanner(file)
	for b.Scan() {
		st.updateMetricFromJSON(b.Bytes())
	}
	log.Println("DOWNLOADED FROM: " + st.Path)
	return nil
}

func (st *MetricStorage) UpdateMetricFromJSON(content []byte) error {
	st.Lock()
	defer st.Unlock()
	return st.updateMetricFromJSON(content)
}

func (st *MetricStorage) updateMetricFromJSON(content []byte) error {
	m, err := metric.SetFromJSON(&metric.Metric{}, content)
	if err != nil {
		return err
	}
	st.Metrics[m.ID] = m
	return nil
}

func (st *MetricStorage) UpdateMetricFromStruct(m metric.Metric) error {
	st.Lock()
	defer st.Unlock()
	st.updateMetricFromStruct(m)
	return nil
}

func (st *MetricStorage) updateMetricFromStruct(m metric.Metric) {
	st.Metrics[m.ID] = m
}

func (st *MetricStorage) MetricExists(mName, mType string) (bool, error) {
	st.Lock()
	defer st.Unlock()
	if m, ok := st.Metrics[mName]; ok {
		if m.MType == mType {
			return true, nil
		}
	}
	return false, nil
}

func (st *MetricStorage) NewMetric(mName, mType, hashKey string, value *float64, delta *int64) error {
	st.Lock()
	defer st.Unlock()
	return st.newMetric(mName, mType, hashKey, value, delta)
}

func (st *MetricStorage) newMetric(mName, mType, hashKey string, value *float64, delta *int64) error {
	if _, ok := st.Metrics[mName]; ok {
		err := errors.New("cannot create: metric <" + mName + "> already exists")
		return err
	}
	m := &metric.Metric{
		ID:    mName,
		MType: mType,
		Value: value,
		Delta: delta,
	}
	m.UpdateHash(hashKey)
	st.updateMetricFromStruct(*m)
	return nil
}

func (st *MetricStorage) GetMetric(name string) (metric.Metric, error) {
	st.Lock()
	defer st.Unlock()
	if m, ok := st.Metrics[name]; ok {
		return m, nil
	}
	err := errors.New("cannot get: metric <" + name + "> doesn't exist")
	return metric.Metric{}, err
}

func (st *MetricStorage) UpdateValue(name, hashKey string, val float64) error {
	st.Lock()
	defer st.Unlock()
	return st.updateValue(name, hashKey, val)
}

func (st *MetricStorage) updateValue(name, hashKey string, val float64) error {
	if m, ok := st.Metrics[name]; ok {
		m.Value = &val
		err := m.UpdateHash(hashKey)
		if err != nil {
			return err
		}
		st.Metrics[name] = m
		return nil
	}
	err := errors.New("cannot update Value: metric <" + name + "> doesn't exist")
	return err
}

func (st *MetricStorage) UpdateDelta(name, hashKey string, val int64) error {
	st.Lock()
	defer st.Unlock()
	return st.updateDelta(name, hashKey, val)
}

func (st *MetricStorage) updateDelta(name, hashKey string, val int64) error {
	if m, ok := st.Metrics[name]; ok {
		m.Delta = &val
		err := m.UpdateHash(hashKey)
		if err != nil {
			return err
		}
		st.Metrics[name] = m
		return nil
	}
	err := errors.New("cannot update Delta: metric <" + name + "> doesn't exist")
	return err
}

func (st *MetricStorage) AddDelta(name, hashKey string, val int64) error {
	st.Lock()
	defer st.Unlock()
	currentVal := st.Metrics[name].Delta
	if currentVal == nil {
		return st.updateDelta(name, hashKey, val)
	}
	return st.updateDelta(name, hashKey, *currentVal+val)
}

func (st *MetricStorage) IncreaseDelta(name, hashKey string) error {
	return st.AddDelta(name, hashKey, 1)
}

func (st *MetricStorage) ResetDelta(name, hashKey string) error {
	st.Lock()
	defer st.Unlock()
	return st.updateDelta(name, hashKey, 0)
}
