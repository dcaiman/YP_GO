package metric

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"

	"github.com/dcaiman/YP_GO/internal/clog"
)

const Schema = `
	(
		mname CHARACTER VARYING PRIMARY KEY,
		mtype CHARACTER VARYING,
		mval DOUBLE PRECISION,
		mdel BIGINT,
		mhash CHARACTER VARYING
	)`

type MStorage interface {
	NewMetric(m Metric) error
	GetMetric(name string) (Metric, error)
	GetAllMetrics() ([]Metric, error)

	MetricExists(name string) (bool, error)
	AccessCheck(ctx context.Context) error

	UpdateBatch(r io.Reader) error
	UpdateMetricFromJSON(content []byte) error
	UpdateMetricFromStruct(m Metric) error

	UpdateValue(name string, val float64) error
	UpdateDelta(name string, del int64) error
	AddDelta(name string, del int64) error
	IncreaseDelta(name string) error
	ResetDelta(name string) error

	DownloadStorage() error
	UploadStorage() error
}

type Metric struct {
	ID    string   `json:"id"`
	MType string   `json:"type"`
	Delta *int64   `json:"delta,omitempty"`
	Value *float64 `json:"value,omitempty"`
	Hash  string   `json:"hash,omitempty"`
}

func (m *Metric) UpdateHash(key string) error {
	if key == "" {
		m.Hash = ""
		return nil
	}

	var deltaPart, valuePart string
	if m.Delta != nil {
		deltaPart = fmt.Sprintf("%s:%s:%d", m.ID, m.MType, *m.Delta)
	}
	if m.Value != nil {
		valuePart = fmt.Sprintf("%s:%s:%f", m.ID, m.MType, *m.Value)
	}

	h := hmac.New(sha256.New, []byte(key))
	_, err := h.Write([]byte(deltaPart + valuePart))
	if err != nil {
		return clog.ToLog(clog.FuncName(), err)
	}
	m.Hash = hex.EncodeToString(h.Sum(nil))
	return nil
}

func (m Metric) GetJSON() ([]byte, error) {
	mj, err := json.Marshal(m)
	if err != nil {
		return []byte{}, clog.ToLog(clog.FuncName(), err)
	}
	return mj, nil
}

func (m *Metric) SetFromJSON(content []byte) error {
	if err := json.Unmarshal(content, m); err != nil {
		return clog.ToLog(clog.FuncName(), err)
	}
	return nil
}
