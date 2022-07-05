package pgxstorage

import (
	"context"
	"database/sql"
	"errors"
	"sync"

	"github.com/dcaiman/YP_GO/internal/metric"
)

type MetricStorage struct {
	sync.RWMutex
	DB      *sql.DB
	HashKey string
	Addr    string
}

func New(dbAddr, hashKey string) (*MetricStorage, error) {
	tmpDB, err := sql.Open("pgx", dbAddr)
	if err != nil {
		return nil, err
	}
	ms := &MetricStorage{
		HashKey: hashKey,
		DB:      tmpDB,
		Addr:    dbAddr,
	}

	exists, err := ms.tableExists()
	if err != nil {
		return nil, err
	}
	if exists {
		return ms, nil
	}
	if err = ms.tableCreate(); err != nil {
		return nil, err
	}
	return ms, nil
}

func (st *MetricStorage) Close() {
	st.Lock()
	defer st.Unlock()
	st.DB.Close()
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
	_, err = st.DB.Exec(`
	INSERT INTO metrics VALUES 
	($1, $2, $3, $4, $5)`, m.ID, m.MType, m.Value, m.Delta, m.Hash)
	if err != nil {
		return err
	}
	return nil
}

func (st *MetricStorage) GetMetric(name string) (metric.Metric, error) {
	st.Lock()
	defer st.Unlock()
	return st.getMetric(name)
}

func (st *MetricStorage) getMetric(name string) (metric.Metric, error) {
	exists, err := st.metricExists(name)
	if err != nil {
		return metric.Metric{}, err
	}
	if !exists {
		err := errors.New("cannot get: metric <" + name + "> doesn't exist")
		return metric.Metric{}, err
	}
	rows, err := st.DB.Query(`SELECT * FROM metrics WHERE mname = $1`, name)
	if err != nil {
		return metric.Metric{}, err
	}
	defer rows.Close()

	m := metric.Metric{}
	for rows.Next() {
		if err := rows.Scan(&m.ID, &m.MType, &m.Value, &m.Delta, &m.Hash); err != nil {
			return metric.Metric{}, err
		}
	}
	if err := rows.Err(); err != nil {
		return metric.Metric{}, err
	}
	return m, nil
}

func (st *MetricStorage) GetAllMetrics() ([]metric.Metric, error) {
	st.Lock()
	defer st.Unlock()
	rows, err := st.DB.Query(`SELECT * FROM metrics`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	allMetrics := []metric.Metric{}
	for rows.Next() {
		m := metric.Metric{}
		if err := rows.Scan(&m.ID, &m.MType, &m.Value, &m.Delta, &m.Hash); err != nil {
			return nil, err
		}
		allMetrics = append(allMetrics, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err

	}
	return allMetrics, nil
}

func (st *MetricStorage) MetricExists(name string) (bool, error) {
	st.Lock()
	defer st.Unlock()
	return st.metricExists(name)
}

func (st *MetricStorage) metricExists(name string) (bool, error) {
	var rows *sql.Rows
	var err error
	exists := false

	rows, err = st.DB.Query(`SELECT EXISTS(SELECT 1 FROM metrics WHERE mname = $1)`, name)
	if err != nil {
		return exists, err
	}
	defer rows.Close()

	for rows.Next() {
		if err := rows.Scan(&exists); err != nil {
			return exists, err
		}
	}
	if err := rows.Err(); err != nil {
		return exists, err

	}
	return exists, nil
}

func (st *MetricStorage) AccessCheck(ctx context.Context) error {
	st.Lock()
	defer st.Unlock()
	if err := st.DB.PingContext(ctx); err != nil {
		return err
	}
	return nil
}

func (st *MetricStorage) UpdateMetricFromJSON(content []byte) error {
	st.Lock()
	defer st.Unlock()
	m := metric.Metric{}
	err := m.SetFromJSON(content)
	if err != nil {
		return err
	}
	return st.updateMetricFromStruct(m)
}

func (st *MetricStorage) UpdateMetricFromStruct(m metric.Metric) error {
	st.Lock()
	defer st.Unlock()
	return st.updateMetricFromStruct(m)
}

func (st *MetricStorage) updateMetricFromStruct(m metric.Metric) error {
	exists, err := st.metricExists(m.ID)
	if err != nil {
		return err
	}
	if exists {
		_, err := st.DB.Exec(`
		UPDATE metrics
		SET mtype = $2, mval = $3, mdel = $4, mhash = $5
		WHERE mname = $1`, m.ID, m.MType, m.Value, m.Delta, m.Hash)
		if err != nil {
			return err
		}
		return nil
	}
	if err := st.newMetric(m); err != nil {
		return err
	}
	return nil
}

func (st *MetricStorage) UpdateValue(name string, val float64) error {
	st.Lock()
	defer st.Unlock()
	m, err := st.getMetric(name)
	if err != nil {
		return err
	}
	m.Value = &val
	if err := m.UpdateHash(st.HashKey); err != nil {
		return err
	}
	err = st.updateMetricFromStruct(m)
	if err != nil {
		return err
	}
	return nil
}

func (st *MetricStorage) UpdateDelta(name string, del int64) error {
	st.Lock()
	defer st.Unlock()
	return st.updateDelta(name, del)
}

func (st *MetricStorage) updateDelta(name string, del int64) error {
	m, err := st.getMetric(name)
	if err != nil {
		return err
	}
	m.Delta = &del
	if err := m.UpdateHash(st.HashKey); err != nil {
		return err
	}
	if err := st.updateMetricFromStruct(m); err != nil {
		return err
	}
	return nil
}

func (st *MetricStorage) AddDelta(name string, del int64) error {
	st.Lock()
	defer st.Unlock()
	return st.addDelta(name, del)
}

func (st *MetricStorage) addDelta(name string, del int64) error {
	m, err := st.getMetric(name)
	if err != nil {
		return err
	}
	newVal := *m.Delta + del
	m.Delta = &newVal
	if err := m.UpdateHash(st.HashKey); err != nil {
		return err
	}
	if err = st.updateMetricFromStruct(m); err != nil {
		return err
	}
	return nil
}

func (st *MetricStorage) IncreaseDelta(name string) error {
	st.Lock()
	defer st.Unlock()
	return st.addDelta(name, 1)
}

func (st *MetricStorage) ResetDelta(name string) error {
	st.Lock()
	defer st.Unlock()
	return st.updateDelta(name, 0)
}

func (st *MetricStorage) tableCreate() error {
	_, err := st.DB.Exec(`CREATE TABLE metrics` + metric.Schema)
	if err != nil {
		return err
	}
	return nil
}

func (st *MetricStorage) tableExists() (bool, error) {
	exists := false
	rows, err := st.DB.Query(`
	SELECT EXISTS(SELECT table_name FROM information_schema.columns WHERE table_name = 'metrics')`)
	if err != nil {
		return exists, err
	}
	for rows.Next() {
		if err := rows.Scan(&exists); err != nil {
			return exists, err
		}
	}
	if rows.Err() != nil {
		return exists, err
	}
	return exists, nil
}

func (st *MetricStorage) DownloadStorage() error {
	return nil
}

func (st *MetricStorage) UploadStorage() error {
	return nil
}
