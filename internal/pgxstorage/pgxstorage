package pgxstorage

import (
	"bufio"
	"context"
	"database/sql"
	"errors"
	"io"
	"sync"

	"github.com/dcaiman/YP_GO/internal/clog"
	"github.com/dcaiman/YP_GO/internal/custom"
	"github.com/dcaiman/YP_GO/internal/metric"
)

const (
	stInsert = `
	INSERT INTO metrics 
	VALUES ($1, $2, $3, $4, $5)`

	stUpdate = `
	UPDATE metrics
	SET mtype = $2, mval = $3, mdel = $4, mhash = $5
	WHERE mname = $1`

	stGetMetric = `
	SELECT * 
	FROM metrics 
	WHERE mname = $1`

	stGetAllMetrics = `
	SELECT * 
	FROM metrics`

	stMetricExists = `
	SELECT 
	EXISTS
	(
		SELECT 1 FROM metrics WHERE mname = $1
	)`

	stTabeExists = `
	SELECT 
	EXISTS
	(
		SELECT table_name 
		FROM information_schema.columns 
		WHERE table_name = 'metrics'
	)`

	stCreateTable = `
	CREATE TABLE metrics`
)

type MetricStorage struct {
	sync.RWMutex
	DB      *sql.DB
	HashKey string
}

func New(dbAddr, hashKey string, drop bool) (*MetricStorage, error) {
	tmpDB, err := sql.Open("pgx", dbAddr)
	if err != nil {
		return nil, clog.ToLog(clog.FuncName(), err)
	}
	ms := &MetricStorage{
		HashKey: hashKey,
		DB:      tmpDB,
	}

	if drop {
		if err := ms.tableDrop(); err != nil {
			return nil, clog.ToLog(clog.FuncName(), err)
		}
	} else {
		exists, err := ms.tableExists()
		if err != nil {
			return nil, clog.ToLog(clog.FuncName(), err)
		}

		if exists {
			return ms, nil
		}
	}

	if err = ms.tableCreate(); err != nil {
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

func (st *MetricStorage) NewMetric(m metric.Metric) error {
	st.Lock()
	defer st.Unlock()

	if err := st.newMetric(m); err != nil {
		return clog.ToLog(clog.FuncName(), err)
	}
	return nil
}

func (st *MetricStorage) newMetric(m metric.Metric) error {
	exists, err := st.metricExists(m.ID)
	if err != nil {
		return clog.ToLog(clog.FuncName(), err)
	}
	if exists {
		return clog.ToLog(clog.FuncName(), errors.New("cannot create: metric <"+m.ID+"> already exists"))
	}
	_, err = st.DB.Exec(stInsert, m.ID, m.MType, m.Value, m.Delta, m.Hash)
	if err != nil {
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
	exists, err := st.metricExists(name)
	if err != nil {
		return metric.Metric{}, clog.ToLog(clog.FuncName(), err)
	}
	if !exists {
		return metric.Metric{}, clog.ToLog(clog.FuncName(), errors.New("cannot get: metric <"+name+"> doesn't exist"))
	}
	rows, err := st.DB.Query(stGetMetric, name)
	if err != nil {
		return metric.Metric{}, clog.ToLog(clog.FuncName(), err)
	}
	defer rows.Close()

	m := metric.Metric{}
	for rows.Next() {
		if err := rows.Scan(&m.ID, &m.MType, &m.Value, &m.Delta, &m.Hash); err != nil {
			return metric.Metric{}, clog.ToLog(clog.FuncName(), err)
		}
	}
	if err := rows.Err(); err != nil {
		return metric.Metric{}, clog.ToLog(clog.FuncName(), err)
	}
	return m, nil
}

func (st *MetricStorage) GetAllMetrics() ([]metric.Metric, error) {
	st.Lock()
	defer st.Unlock()

	rows, err := st.DB.Query(stGetAllMetrics)
	if err != nil {
		return nil, clog.ToLog(clog.FuncName(), err)
	}
	defer rows.Close()

	allMetrics := []metric.Metric{}
	for rows.Next() {
		m := metric.Metric{}
		if err := rows.Scan(&m.ID, &m.MType, &m.Value, &m.Delta, &m.Hash); err != nil {
			return nil, clog.ToLog(clog.FuncName(), err)
		}
		allMetrics = append(allMetrics, m)
	}
	if err := rows.Err(); err != nil {
		return nil, clog.ToLog(clog.FuncName(), err)

	}
	return allMetrics, nil
}

func (st *MetricStorage) MetricExists(name string) (bool, error) {
	st.Lock()
	defer st.Unlock()

	exists, err := st.metricExists(name)
	if err != nil {
		return false, clog.ToLog(clog.FuncName(), err)
	}
	return exists, nil
}

func (st *MetricStorage) metricExists(name string) (bool, error) {
	var rows *sql.Rows
	var err error
	exists := false

	rows, err = st.DB.Query(stMetricExists, name)
	if err != nil {
		return exists, clog.ToLog(clog.FuncName(), err)
	}
	defer rows.Close()

	for rows.Next() {
		if err := rows.Scan(&exists); err != nil {
			return exists, clog.ToLog(clog.FuncName(), err)
		}
	}
	if err := rows.Err(); err != nil {
		return exists, clog.ToLog(clog.FuncName(), err)

	}
	return exists, nil
}

func (st *MetricStorage) AccessCheck(ctx context.Context) error {
	st.Lock()
	defer st.Unlock()

	if err := st.DB.PingContext(ctx); err != nil {
		return clog.ToLog(clog.FuncName(), err)
	}
	return nil
}

func (st *MetricStorage) UpdateMetricFromJSON(content []byte) error {
	st.Lock()
	defer st.Unlock()

	m := metric.Metric{}
	if err := m.SetFromJSON(content); err != nil {
		return clog.ToLog(clog.FuncName(), err)
	}
	if err := st.updateMetricFromStruct(m); err != nil {
		return clog.ToLog(clog.FuncName(), err)
	}
	return nil
}

func (st *MetricStorage) UpdateMetricFromStruct(m metric.Metric) error {
	st.Lock()
	defer st.Unlock()

	if err := st.updateMetricFromStruct(m); err != nil {
		return clog.ToLog(clog.FuncName(), err)
	}
	return nil
}

func (st *MetricStorage) updateMetricFromStruct(m metric.Metric) error {
	exists, err := st.metricExists(m.ID)
	if err != nil {
		return clog.ToLog(clog.FuncName(), err)
	}
	if exists {
		_, err := st.DB.Exec(stUpdate, m.ID, m.MType, m.Value, m.Delta, m.Hash)
		if err != nil {
			return clog.ToLog(clog.FuncName(), err)
		}
		return nil
	}
	if err := st.newMetric(m); err != nil {
		return clog.ToLog(clog.FuncName(), err)
	}
	return nil
}

func (st *MetricStorage) UpdateBatch(r io.Reader) error {
	st.Lock()
	defer st.Unlock()

	tx, err := st.DB.Begin()
	if err != nil {
		return clog.ToLog(clog.FuncName(), err)
	}
	defer tx.Rollback()

	txStGet, err := tx.Prepare(stGetMetric)
	if err != nil {
		return clog.ToLog(clog.FuncName(), err)
	}
	defer txStGet.Close()

	txStUpdate, err := tx.Prepare(stUpdate)
	if err != nil {
		return clog.ToLog(clog.FuncName(), err)
	}
	defer txStUpdate.Close()

	txStInsert, err := tx.Prepare(stInsert)
	if err != nil {
		return clog.ToLog(clog.FuncName(), err)
	}
	defer txStInsert.Close()

	txStExist, err := tx.Prepare(stMetricExists)
	if err != nil {
		return clog.ToLog(clog.FuncName(), err)
	}
	defer txStExist.Close()

	s := bufio.NewScanner(r)
	s.Split(custom.CustomSplit())
	for s.Scan() {
		m := metric.Metric{}
		if err := m.SetFromJSON(s.Bytes()); err != nil {
			return clog.ToLog(clog.FuncName(), err)
		}

		//EXIST BEGIN
		exists := false
		rows, err := txStExist.Query(m.ID)
		if err != nil {
			return clog.ToLog(clog.FuncName(), err)
		}
		for rows.Next() {
			if err := rows.Scan(&exists); err != nil {
				return clog.ToLog(clog.FuncName(), err)
			}
		}
		if err := rows.Err(); err != nil {
			return clog.ToLog(clog.FuncName(), err)
		}
		rows.Close()
		//EXIST END

		if exists {
			if m.Delta != nil {
				tmp := *m.Delta

				//GET BEGIN
				rows, err := txStGet.Query(m.ID)
				if err != nil {
					return clog.ToLog(clog.FuncName(), err)
				}
				mTmp := metric.Metric{}
				for rows.Next() {
					if err := rows.Scan(&mTmp.ID, &mTmp.MType, &mTmp.Value, &mTmp.Delta, &mTmp.Hash); err != nil {
						return clog.ToLog(clog.FuncName(), err)
					}
				}
				if err := rows.Err(); err != nil {
					return clog.ToLog(clog.FuncName(), err)
				}
				rows.Close()
				//GET END

				tmp += *mTmp.Delta
				m.Delta = &tmp
			}
			if _, err := txStUpdate.Exec(m.ID, m.MType, m.Value, m.Delta, m.Hash); err != nil {
				return clog.ToLog(clog.FuncName(), err)
			}
		} else if _, err := txStInsert.Exec(m.ID, m.MType, m.Value, m.Delta, m.Hash); err != nil {
			return clog.ToLog(clog.FuncName(), err)
		}
	}
	if err := tx.Commit(); err != nil {
		return clog.ToLog(clog.FuncName(), err)
	}
	return nil
}

func (st *MetricStorage) UpdateValue(name string, val float64) error {
	st.Lock()
	defer st.Unlock()

	m, err := st.getMetric(name)
	if err != nil {
		return clog.ToLog(clog.FuncName(), err)
	}
	m.Value = &val
	if err := m.UpdateHash(st.HashKey); err != nil {
		return clog.ToLog(clog.FuncName(), err)
	}
	if err := st.updateMetricFromStruct(m); err != nil {
		return clog.ToLog(clog.FuncName(), err)
	}
	return nil
}

func (st *MetricStorage) UpdateDelta(name string, del int64) error {
	st.Lock()
	defer st.Unlock()

	if err := st.updateDelta(name, del); err != nil {
		return clog.ToLog(clog.FuncName(), err)
	}
	return nil
}

func (st *MetricStorage) updateDelta(name string, del int64) error {
	m, err := st.getMetric(name)
	if err != nil {
		return clog.ToLog(clog.FuncName(), err)
	}
	m.Delta = &del
	if err := m.UpdateHash(st.HashKey); err != nil {
		return clog.ToLog(clog.FuncName(), err)
	}
	if err := st.updateMetricFromStruct(m); err != nil {
		return clog.ToLog(clog.FuncName(), err)
	}
	return nil
}

func (st *MetricStorage) AddDelta(name string, del int64) error {
	st.Lock()
	defer st.Unlock()

	if err := st.addDelta(name, del); err != nil {
		return clog.ToLog(clog.FuncName(), err)
	}
	return nil
}

func (st *MetricStorage) addDelta(name string, del int64) error {
	m, err := st.getMetric(name)
	if err != nil {
		return clog.ToLog(clog.FuncName(), err)
	}
	newVal := *m.Delta + del
	m.Delta = &newVal
	if err := m.UpdateHash(st.HashKey); err != nil {
		return clog.ToLog(clog.FuncName(), err)
	}
	if err = st.updateMetricFromStruct(m); err != nil {
		return clog.ToLog(clog.FuncName(), err)
	}
	return nil
}

func (st *MetricStorage) IncreaseDelta(name string) error {
	st.Lock()
	defer st.Unlock()

	if err := st.addDelta(name, 1); err != nil {
		return clog.ToLog(clog.FuncName(), err)
	}
	return nil
}

func (st *MetricStorage) ResetDelta(name string) error {
	st.Lock()
	defer st.Unlock()

	if err := st.updateDelta(name, 0); err != nil {
		return clog.ToLog(clog.FuncName(), err)
	}
	return nil
}

func (st *MetricStorage) tableCreate() error {
	_, err := st.DB.Exec(stCreateTable + metric.Schema)
	if err != nil {
		return clog.ToLog(clog.FuncName(), err)
	}
	return nil
}

func (st *MetricStorage) tableExists() (bool, error) {
	exists := false
	rows, err := st.DB.Query(stTabeExists)
	if err != nil {
		return false, clog.ToLog(clog.FuncName(), err)
	}
	for rows.Next() {
		if err := rows.Scan(&exists); err != nil {
			return false, clog.ToLog(clog.FuncName(), err)
		}
	}
	if rows.Err() != nil {
		return false, clog.ToLog(clog.FuncName(), err)
	}
	return exists, nil
}

func (st *MetricStorage) DownloadStorage() error {
	return nil
}

func (st *MetricStorage) UploadStorage() error {
	return nil
}

func (st *MetricStorage) tableDrop() error {
	_, err := st.DB.Exec(`DROP TABLE metrics`)
	if err != nil {
		return clog.ToLog(clog.FuncName(), err)
	}
	return nil
}
