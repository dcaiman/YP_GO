package metrics

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

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
