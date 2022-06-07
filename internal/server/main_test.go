package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_updGauge(t *testing.T) {
	var expected float64 = 5
	storage.updateGauge("test", expected)
	assert.Equal(t, storage.Gauges["test"], expected)
}

func Test_updCounter(t *testing.T) {
	var expected int64 = 5
	storage.updateCounter("new", expected)
	assert.Equal(t, storage.Counters["new"], expected)
}
