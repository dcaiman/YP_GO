package pgxstorage

import (
	"context"
	"database/sql"
	"errors"
	"sync"

	"github.com/dcaiman/YP_GO/internal/clog"
	"github.com/dcaiman/YP_GO/internal/metric"
)

const (
	stUpdateMetric = `
	INSERT INTO metrics
	VALUES ($1, $2, $3, $4) 
	ON CONFLICT (mname)
	DO
	UPDATE
	SET mtype = $2, mval = $3, mdel = metrics.mdel + $4`

	stGetMetric = `
	SELECT * 
	FROM metrics 
	WHERE mname = $1`

	stGetBatch = `
	SELECT * 
	FROM metrics`

	stCreateTableIfNotExists = `
	CREATE TABLE IF NOT EXISTS metrics ` +
		metric.Schema

	stDropTableIfExisis = `
	DROP TABLE IF EXISTS metrics`
)

type MetricStorage struct {
	sync.RWMutex
	DB *sql.DB
}

func New(dbAddr string, drop bool) (*MetricStorage, error) {
	tmpDB, err := sql.Open("pgx", dbAddr)
	if err != nil {
		return nil, clog.ToLog(clog.FuncName(), err)
	}
	ms := &MetricStorage{
		DB: tmpDB,
	}

	if drop {
		_, err := ms.DB.Exec(stDropTableIfExisis)
		if err != nil {
			return nil, clog.ToLog(clog.FuncName(), err)
		}
	}
	_, err = ms.DB.Exec(stCreateTableIfNotExists)
	if err != nil {
		return nil, clog.ToLog(clog.FuncName(), err)
	}
	return ms, nil
}

func (st *MetricStorage) Close() error {
	st.Lock()
	defer st.Unlock()

	if err := st.DB.Close(); err != nil {
		return clog.ToLog(clog.FuncName(), err)
	}
	return nil
}

func (st *MetricStorage) AccessCheck(ctx context.Context) error {
	st.Lock()
	defer st.Unlock()

	if err := st.DB.PingContext(ctx); err != nil {
		return clog.ToLog(clog.FuncName(), err)
	}
	return nil
}

func (st *MetricStorage) GetMetric(name string) (metric.Metric, error) {
	st.Lock()
	defer st.Unlock()

	m, err := st.getMetric(name)
	if err != nil {
		return m, clog.ToLog(clog.FuncName(), err)
	}
	return m, nil
}

func (st *MetricStorage) getMetric(name string) (metric.Metric, error) {
	rows, err := st.DB.Query(stGetMetric, name)
	if err != nil {
		return metric.Metric{}, clog.ToLog(clog.FuncName(), err)
	}
	defer rows.Close()

	m := metric.Metric{}
	for rows.Next() {
		if err := rows.Scan(&m.ID, &m.MType, &m.Value, &m.Delta); err != nil {
			return metric.Metric{}, clog.ToLog(clog.FuncName(), err)
		}
	}
	if m.ID == "" {
		return metric.Metric{}, clog.ToLog(clog.FuncName(), errors.New("cannto get: metric <"+name+"> doesn't exist"))
	}
	if err := rows.Err(); err != nil {
		return metric.Metric{}, clog.ToLog(clog.FuncName(), err)
	}
	return m, nil
}

func (st *MetricStorage) GetBatch() ([]metric.Metric, error) {
	st.Lock()
	defer st.Unlock()

	rows, err := st.DB.Query(stGetBatch)
	if err != nil {
		return nil, clog.ToLog(clog.FuncName(), err)
	}
	defer rows.Close()

	allMetrics := []metric.Metric{}
	for rows.Next() {
		m := metric.Metric{}
		if err := rows.Scan(&m.ID, &m.MType, &m.Value, &m.Delta); err != nil {
			return nil, clog.ToLog(clog.FuncName(), err)
		}
		allMetrics = append(allMetrics, m)
	}
	if err := rows.Err(); err != nil {
		return nil, clog.ToLog(clog.FuncName(), err)

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
	if _, err := st.DB.Exec(stUpdateMetric, m.ID, m.MType, m.Value, m.Delta); err != nil {
		return clog.ToLog(clog.FuncName(), err)
	}
	return nil
}

func (st *MetricStorage) UpdateBatch(batch []metric.Metric) error {
	st.Lock()
	defer st.Unlock()

	tx, err := st.DB.Begin()
	if err != nil {
		return clog.ToLog(clog.FuncName(), err)
	}
	defer tx.Rollback()

	txStUpdateMetric, err := tx.Prepare(stUpdateMetric)
	if err != nil {
		return clog.ToLog(clog.FuncName(), err)
	}
	defer txStUpdateMetric.Close()

	for i := range batch {
		if _, err := txStUpdateMetric.Exec(batch[i].ID, batch[i].MType, batch[i].Value, batch[i].Delta); err != nil {
			return clog.ToLog(clog.FuncName(), err)
		}
	}
	if err := tx.Commit(); err != nil {
		return clog.ToLog(clog.FuncName(), err)
	}
	return nil
}
