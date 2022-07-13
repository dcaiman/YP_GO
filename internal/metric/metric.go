package metric

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

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
	GetMetric(id string) (Metric, error)
	GetBatch() ([]Metric, error)

	UpdateMetric(m Metric) error
	UpdateBatch(batch []Metric) error

	AccessCheck(ctx context.Context) error
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
