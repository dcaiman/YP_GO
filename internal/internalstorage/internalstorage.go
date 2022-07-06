package internalstorage

import (
	"bufio"
	"context"
	"errors"
	"io"
	"log"
	"os"
	"sync"

	"github.com/dcaiman/YP_GO/internal/metric"
)

type MetricStorage struct {
	sync.RWMutex
	HashKey  string
	FilePath string
	Metrics  map[string]metric.Metric
}

func New(filePath, hashKey string) *MetricStorage {
	ms := &MetricStorage{
		HashKey:  hashKey,
		FilePath: filePath,
		Metrics:  map[string]metric.Metric{},
	}
	return ms
}

func (st *MetricStorage) NewMetric(m metric.Metric) error {
	st.Lock()
	defer st.Unlock()
	return st.newMetric(m)
}

func (st *MetricStorage) newMetric(m metric.Metric) error {
	exists, err := st.metricExists(m.ID)
	if err != nil {
		return err
	}
	if exists {
		err := errors.New("cannot create: metric <" + m.ID + "> already exists")
		return err
	}
	st.updateMetricFromStruct(m)
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

func (st *MetricStorage) GetAllMetrics() ([]metric.Metric, error) {
	allMetrics := []metric.Metric{}
	for k := range st.Metrics {
		allMetrics = append(allMetrics, st.Metrics[k])
	}
	return allMetrics, nil
}

func (st *MetricStorage) MetricExists(name string) (bool, error) {
	st.Lock()
	defer st.Unlock()
	return st.metricExists(name)
}

func (st *MetricStorage) metricExists(name string) (bool, error) {
	if _, ok := st.Metrics[name]; ok {
		return true, nil
	}
	return false, nil
}

func (st *MetricStorage) AccessCheck(ctx context.Context) error {
	if st.Metrics == nil {
		err := errors.New("storage map is not initialized")
		return err
	}
	return nil
}

func (st *MetricStorage) UpdateMetricFromJSON(content []byte) error {
	st.Lock()
	defer st.Unlock()
	return st.updateMetricFromJSON(content)
}

func (st *MetricStorage) updateMetricFromJSON(content []byte) error {
	m := metric.Metric{}
	err := m.SetFromJSON(content)
	if err != nil {
		return err
	}
	st.updateMetricFromStruct(m)
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

func (st *MetricStorage) UpdateBatch(r io.Reader) error {
	st.Lock()
	defer st.Unlock()

	b := bufio.NewScanner(r)
	for b.Scan() {
		m := metric.Metric{}
		m.SetFromJSON(b.Bytes())
		exists, err := st.metricExists(m.ID)
		if err != nil {
			return err
		}
		if exists && m.Delta != nil {
			tmp := *m.Delta
			tmp += *st.Metrics[m.ID].Delta
			m.Delta = &tmp
		}
		st.updateMetricFromStruct(m)
	}
	return nil
}

func (st *MetricStorage) UpdateValue(name string, val float64) error {
	st.Lock()
	defer st.Unlock()

	return st.updateValue(name, val)
}

func (st *MetricStorage) updateValue(name string, val float64) error {
	if m, ok := st.Metrics[name]; ok {
		m.Value = &val
		if err := m.UpdateHash(st.HashKey); err != nil {
			return err
		}
		st.Metrics[name] = m
		return nil
	}
	err := errors.New("cannot update Value: metric <" + name + "> doesn't exist")
	return err
}

func (st *MetricStorage) UpdateDelta(name string, del int64) error {
	st.Lock()
	defer st.Unlock()
	return st.updateDelta(name, del)
}

func (st *MetricStorage) updateDelta(name string, del int64) error {
	if m, ok := st.Metrics[name]; ok {
		m.Delta = &del
		if err := m.UpdateHash(st.HashKey); err != nil {
			return err
		}
		st.Metrics[name] = m
		return nil
	}
	err := errors.New("cannot update Delta: metric <" + name + "> doesn't exist")
	return err
}

func (st *MetricStorage) AddDelta(name string, del int64) error {
	st.Lock()
	defer st.Unlock()
	currentDel := st.Metrics[name].Delta
	if currentDel == nil {
		return st.updateDelta(name, del)
	}
	return st.updateDelta(name, *currentDel+del)
}

func (st *MetricStorage) IncreaseDelta(name string) error {
	return st.AddDelta(name, 1)
}

func (st *MetricStorage) ResetDelta(name string) error {
	st.Lock()
	defer st.Unlock()
	return st.updateDelta(name, 0)
}

func (st *MetricStorage) DownloadStorage() error {
	st.Lock()
	defer st.Unlock()
	file, err := os.Open(st.FilePath)
	if err != nil {
		return err
	}
	defer file.Close()
	b := bufio.NewScanner(file)
	for b.Scan() {
		st.updateMetricFromJSON(b.Bytes())
	}
	log.Println("DOWNLOADED FROM: " + st.FilePath)
	return nil
}

func (st *MetricStorage) UploadStorage() error {
	st.Lock()
	defer st.Unlock()
	file, err := os.OpenFile(st.FilePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0777)
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
	log.Println("UPLOADED TO: " + st.FilePath)
	return nil
}
