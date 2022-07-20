package internalstorage

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"log"
	"os"
	"sync"

	"github.com/dcaiman/YP_GO/internal/clog"
	"github.com/dcaiman/YP_GO/internal/metric"
)

type MetricStorage struct {
	sync.RWMutex
	FilePath string
	Metrics  map[string]metric.Metric
}

func New(filePath string) *MetricStorage {
	ms := &MetricStorage{
		Metrics:  map[string]metric.Metric{},
		FilePath: filePath,
	}
	return ms
}

func (st *MetricStorage) GetMetric(name string) (metric.Metric, error) {
	st.Lock()
	defer st.Unlock()

	if m, ok := st.Metrics[name]; ok {
		return m, nil
	}
	return metric.Metric{}, clog.ToLog(clog.FuncName(), errors.New("cannot get: metric <"+name+"> doesn't exist"))
}

func (st *MetricStorage) GetBatch() ([]metric.Metric, error) {
	st.Lock()
	defer st.Unlock()

	allMetrics := []metric.Metric{}
	for k := range st.Metrics {
		allMetrics = append(allMetrics, st.Metrics[k])
	}
	return allMetrics, nil
}

func (st *MetricStorage) UpdateMetric(m metric.Metric) error {
	st.Lock()
	defer st.Unlock()

	if err := st.updateMetric(m); err != nil {
		return clog.ToLog(clog.FuncName(), err)
	}
	return nil
}

func (st *MetricStorage) updateMetric(m metric.Metric) error {
	if m.Delta != nil {
		if mEx, ok := st.Metrics[m.ID]; ok && mEx.Delta != nil {
			del := *mEx.Delta + *m.Delta
			m.Delta = &del
		}
	}
	st.Metrics[m.ID] = m
	return nil
}

func (st *MetricStorage) UpdateBatch(batch []metric.Metric) error {
	st.Lock()
	defer st.Unlock()

	for i := range batch {
		if err := st.updateMetric(batch[i]); err != nil {
			return clog.ToLog(clog.FuncName(), err)
		}
	}
	return nil
}

func (st *MetricStorage) AccessCheck(ctx context.Context) error {
	st.Lock()
	defer st.Unlock()

	if st.Metrics == nil {
		return clog.ToLog(clog.FuncName(), errors.New("storage map is not initialized"))
	}
	return nil
}

func (st *MetricStorage) UploadStorage() error {
	st.Lock()
	defer st.Unlock()

	file, err := os.OpenFile(st.FilePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0777)
	if err != nil {
		return clog.ToLog(clog.FuncName(), err)
	}
	defer file.Close()
	for name := range st.Metrics {
		mj, err := json.Marshal(st.Metrics[name])
		if err != nil {
			return clog.ToLog(clog.FuncName(), err)
		}
		mj = append(mj, '\n')
		_, err = file.Write(mj)
		if err != nil {
			return clog.ToLog(clog.FuncName(), err)
		}
	}
	log.Println("UPLOADED TO: " + st.FilePath)
	return nil
}

func (st *MetricStorage) DownloadStorage() error {
	st.Lock()
	defer st.Unlock()

	file, err := os.Open(st.FilePath)
	if err != nil {
		return clog.ToLog(clog.FuncName(), err)
	}
	defer file.Close()
	b := bufio.NewScanner(file)
	for b.Scan() {
		m := metric.Metric{}
		if err := json.Unmarshal(b.Bytes(), &m); err != nil {
			return clog.ToLog(clog.FuncName(), err)
		}
		if err := st.updateMetric(m); err != nil {
			return clog.ToLog(clog.FuncName(), err)
		}
	}
	log.Println("DOWNLOADED FROM: " + st.FilePath)
	return nil
}
