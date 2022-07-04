package pgxstorage

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"text/template"

	"github.com/dcaiman/YP_GO/internal/metric"
)

type MetricStorage struct {
	DB   *sql.DB
	Path string
}

func (st *MetricStorage) Init(path string) {
	tmp, err := sql.Open("pgx", path)
	if err != nil {
		log.Println(err.Error())
	}
	st.DB = tmp
	st.Path = path

	if err = st.tableCreate(); err != nil {
		log.Println(err.Error())
	}

}

func (st *MetricStorage) AccessCheck(ctx context.Context) error {
	if err := st.DB.PingContext(ctx); err != nil {
		return err
	}
	return nil
}

func (st *MetricStorage) GetHTML() (*template.Template, error) {
	html := "METRICS LIST:"
	rows, err := st.DB.Query(`SELECT * FROM metrics `)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	m := metric.Metric{}
	for rows.Next() {
		rows.Scan(&m.ID, &m.MType, &m.Value, &m.Delta, &m.Hash)
		if err != nil {
			return nil, err
		}
		mj, err := m.GetJSON()
		if err != nil {
			return nil, err
		}
		html += "<p>" + string(mj) + "</p>"
	}
	t, err := template.New("").Parse(html)
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (st *MetricStorage) UploadStorage() error {
	log.Println("NO NEED TO UPLOAD USING DB")
	return nil
}

func (st *MetricStorage) DownloadStorage() error {
	log.Println("NO NEED TO DOWNLOAD USING DB")
	return nil
}

func (st *MetricStorage) UpdateMetricFromJSON(content []byte) error {
	m, err := metric.SetFromJSON(&metric.Metric{}, content)
	if err != nil {
		return err
	}
	return st.UpdateMetricFromStruct(m)
}

func (st *MetricStorage) UpdateMetricFromStruct(m metric.Metric) error {
	exists, err := st.MetricExists(m.ID, m.MType)
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
	_, err = st.DB.Exec(`
	INSERT INTO metrics VALUES 
	($1, $2, $3, $4, $5)`, m.ID, m.MType, m.Value, m.Delta, m.Hash)
	if err != nil {
		return err
	}
	return nil
}

func (st *MetricStorage) MetricExists(mName, mType string) (bool, error) {
	exists := false
	rows, err := st.DB.Query(`
	SELECT EXISTS(SELECT 1 FROM metrics WHERE mname = $1 AND mtype = $2)`, mName, mType)
	if err != nil {
		return exists, err
	}
	defer rows.Close()

	for rows.Next() {
		rows.Scan(&exists)
		if err != nil {
			return exists, err
		}
	}
	return exists, nil
}

func (st *MetricStorage) NewMetric(mName, mType, hashKey string, value *float64, delta *int64) error {
	m := &metric.Metric{
		ID:    mName,
		MType: mType,
		Value: value,
		Delta: delta,
	}
	m.UpdateHash(hashKey)

	_, err := st.DB.Exec(`
	INSERT INTO metrics VALUES 
	($1, $2, $3, $4, $5)`, m.ID, m.MType, m.Value, m.Delta, m.Hash)
	if err != nil {
		return err
	}
	return nil
}

func (st *MetricStorage) GetMetric(name string) (metric.Metric, error) {
	rows, err := st.DB.Query(`
	SELECT * FROM metrics WHERE mname = $1`, name)
	if err != nil {
		return metric.Metric{}, err
	}
	defer rows.Close()

	m := metric.Metric{}
	for rows.Next() {
		rows.Scan(&m.ID, &m.MType, &m.Value, &m.Delta, &m.Hash)
		if err != nil {
			return metric.Metric{}, err
		}
	}
	if (m == metric.Metric{}) {
		err := errors.New("cannot get: metric <" + name + "> doesn't exist")
		return metric.Metric{}, err
	}
	return m, nil
}

func (st *MetricStorage) UpdateValue(name, hashKey string, val float64) error {
	m, err := st.GetMetric(name)
	if err != nil {
		return err
	}
	m.Value = &val
	err = m.UpdateHash(hashKey)
	if err != nil {
		return err
	}
	err = st.UpdateMetricFromStruct(m)
	if err != nil {
		return err
	}
	return nil
}

func (st *MetricStorage) UpdateDelta(name, hashKey string, val int64) error {
	m, err := st.GetMetric(name)
	if err != nil {
		return err
	}
	m.Delta = &val
	err = m.UpdateHash(hashKey)
	if err != nil {
		return err
	}
	err = st.UpdateMetricFromStruct(m)
	if err != nil {
		return err
	}
	return nil
}

func (st *MetricStorage) AddDelta(name, hashKey string, val int64) error {
	m, err := st.GetMetric(name)
	if err != nil {
		return err
	}
	newVal := *m.Delta + val
	m.Delta = &newVal
	err = m.UpdateHash(hashKey)
	if err != nil {
		return err
	}
	err = st.UpdateMetricFromStruct(m)
	if err != nil {
		return err
	}
	return nil
}

func (st *MetricStorage) IncreaseDelta(name, hashKey string) error {
	return st.AddDelta(name, hashKey, 1)
}

func (st *MetricStorage) ResetDelta(name, hashKey string) error {
	m, err := st.GetMetric(name)
	if err != nil {
		return err
	}
	var newVal int64 = 0
	m.Delta = &newVal
	err = m.UpdateHash(hashKey)
	if err != nil {
		return err
	}
	err = st.UpdateMetricFromStruct(m)
	if err != nil {
		return err
	}
	return nil
}

func (st *MetricStorage) tableCreate() error {
	exists, err := st.tableExists()
	if err != nil {
		return err
	}
	if exists {
		err := errors.New("table <metrics> already exists")
		return err
	}
	_, err = st.DB.Exec(`CREATE TABLE metrics` + metric.Schema)

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
	} else {
		for rows.Next() {
			rows.Scan(&exists)
			if err != nil {
				return exists, err
			}
		}
	}
	return exists, nil
}

/*
func (st *MetricStorage) tableDrop() error {
	_, err := st.DB.Exec(`DROP TABLE metrics`)
	if err != nil {
		return err
	}
	return nil
}
*/
