package metric

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"text/template"
)

const Schema = `
	(
		mname CHARACTER VARYING PRIMARY KEY,
		mtype CHARACTER VARYING,
		mval DOUBLE PRECISION,
		mdel INTEGER,
		mhash CHARACTER VARYING
	)`

type MStorage interface {
	Init(custom string)
	AccessCheck(ctx context.Context) error
	DownloadStorage() error
	UploadStorage() error
	GetHTML() (*template.Template, error)
	UpdateMetricFromJSON(content []byte) error
	UpdateMetricFromStruct(m Metric) error
	MetricExists(mName, mType string) (bool, error)
	NewMetric(mName, mType, hashKey string, value *float64, delta *int64) error
	GetMetric(name string) (Metric, error)
	UpdateValue(name, hashKey string, val float64) error
	UpdateDelta(name, hashKey string, val int64) error
	AddDelta(name, hashKey string, val int64) error
	IncreaseDelta(name, hashKey string) error
	ResetDelta(name, hashKey string) error
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
		deltaPart = fmt.Sprintf("%s:counter:%d", m.ID, *m.Delta)
	}
	if m.Value != nil {
		valuePart = fmt.Sprintf("%s:gauge:%f", m.ID, *m.Value)
	}

	h := hmac.New(sha256.New, []byte(key))
	_, err := h.Write([]byte(deltaPart + valuePart))
	if err != nil {
		return err
	}
	m.Hash = hex.EncodeToString(h.Sum(nil))
	return nil
}

func (m Metric) GetJSON() ([]byte, error) {
	mj, err := json.Marshal(m)
	if err != nil {
		return []byte{}, err
	}
	return mj, nil
}

func (m *Metric) SetFromJSON(content []byte) (Metric, error) {
	return SetFromJSON(m, content)
}

func SetFromJSON(m *Metric, content []byte) (Metric, error) {
	if err := json.Unmarshal(content, m); err != nil {
		return *m, err
	}
	return *m, nil
}
