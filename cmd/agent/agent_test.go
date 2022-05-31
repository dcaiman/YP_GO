package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_newGauge(t *testing.T) {
	Actual := NewGauge("new")
	Expected := Gauge{Name: "new", Type: "gauge"}
	assert.Equal(t, Expected, Actual)
}

func Test_newCounter(t *testing.T) {
	Actual := NewCounter("new")
	Expected := Counter{Name: "new", Type: "counter"}
	assert.Equal(t, Expected, Actual)
}
