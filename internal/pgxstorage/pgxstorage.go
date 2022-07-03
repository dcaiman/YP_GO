package pgxstorage

import (
	"context"
	"database/sql"
	"log"

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
}

func (st *MetricStorage) AccessCheck(ctx context.Context) error {
	if err := st.DB.PingContext(ctx); err != nil {
		return err
	}
	return nil
}

func (st *MetricStorage) UploadStorage() error {
	return nil
}

func (st *MetricStorage) DownloadStorage() error {
	return nil
}

func (st *MetricStorage) UpdateMetricFromJSON(content []byte) error {
	return nil
}

func (st *MetricStorage) UpdateMetricFromStruct(m metric.Metric) {
}

func (st *MetricStorage) MetricExists(mName, mType string) bool {
	return false
}

func (st *MetricStorage) NewMetric(mName, mType, hashKey string, value *float64, delta *int64) error {
	return nil
}

func (st *MetricStorage) GetMetric(name string) (metric.Metric, error) {
	return metric.Metric{}, nil
}

func (st *MetricStorage) UpdateValue(name, hashKey string, val float64) error {
	return nil
}

func (st *MetricStorage) UpdateDelta(name, hashKey string, val int64) error {
	return nil
}

func (st *MetricStorage) AddDelta(name, hashKey string, val int64) error {
	return nil
}

func (st *MetricStorage) IncreaseDelta(name, hashKey string) error {
	return nil
}

func (st *MetricStorage) ResetDelta(name, hashKey string) error {
	return nil
}
