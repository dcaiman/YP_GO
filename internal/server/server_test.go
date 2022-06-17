package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
)

func Test_updateGauge(t *testing.T) {
	storage = Metrics{
		Gauges: map[string]float64{},
	}
	var expected float64 = 5
	storage.updateGauge("test", expected)
	assert.Equal(t, expected, storage.Gauges["test"])
}

func Test_updateCounter(t *testing.T) {
	storage = Metrics{
		Counters: map[string]int64{},
	}
	var expected int64 = 5
	storage.updateCounter("new", expected)
	assert.Equal(t, expected, storage.Counters["new"])
}

func Test_getGauges(t *testing.T) {
	storage = Metrics{
		Gauges: map[string]float64{
			"g1": 0.5,
			"g2": 5.5,
		},
	}
	expected := []string{
		"g1: 0.500",
		"g2: 5.500",
	}
	assert.ElementsMatch(t, expected, storage.getGauges())
}

func Test_getCounters(t *testing.T) {
	storage = Metrics{
		Counters: map[string]int64{
			"c1": 1,
			"c2": 5,
		},
	}
	expected := []string{
		"c1: 1",
		"c2: 5",
	}
	assert.ElementsMatch(t, expected, storage.getCounters())
}

func Test_getGauge(t *testing.T) {
	var expected = 0.5
	storage = Metrics{
		Gauges: map[string]float64{
			"g1": expected,
		},
	}
	actual, _ := storage.getGauge("g1")
	assert.Equal(t, expected, actual)
}

func Test_getCounter(t *testing.T) {
	var expected int64 = 5
	storage = Metrics{
		Counters: map[string]int64{
			"c1": expected,
		},
	}
	actual, _ := storage.getCounter("c1")
	assert.Equal(t, expected, actual)
}

func Test_handlerGetAll(t *testing.T) {
	storage = Metrics{
		Gauges: map[string]float64{
			"g1": 5.5,
		},
		Counters: map[string]int64{
			"c1": 1,
		},
	}
	expectedBody := "GAUGES LIST:\n\ng1: 5.500\n\nCOUNTERS LIST:\n\nc1: 1"
	expectedStatusCode := 200

	testRouter := chi.NewRouter()
	testRouter.Route("/", func(r chi.Router) {
		r.Get("/", handlerGetAll)
	})
	testServer := httptest.NewServer(testRouter)
	defer testServer.Close()

	res, err := http.Get(testServer.URL)
	assert.NoError(t, err)
	defer res.Body.Close()
	resBody, _ := io.ReadAll(res.Body)

	assert.Equal(t, expectedBody, string(resBody))
	assert.Equal(t, expectedStatusCode, res.StatusCode)
}

func Test_handlerUpdate(t *testing.T) {
	storage = Metrics{
		Gauges: map[string]float64{
			"g1": 5.5,
		},
	}
	tests := []struct {
		name               string
		url                string
		expectedValue      float64
		expectedStatusCode int
	}{
		{
			name:               "#1 existing metric updating",
			url:                "/update/gauge/g1/10.500",
			expectedValue:      10.5,
			expectedStatusCode: 200,
		},
		{
			name:               "#2 unknown path",
			url:                "/updates/gauge/g1/11.500",
			expectedValue:      10.5,
			expectedStatusCode: 404,
		},
		{
			name:               "#3 unknown metric type",
			url:                "/update/gauges/g1/11.500",
			expectedValue:      10.5,
			expectedStatusCode: 501,
		},
		{
			name:               "#4 new metric",
			url:                "/update/gauge/g2/11.500",
			expectedValue:      10.5,
			expectedStatusCode: 200,
		},
	}
	for _, currentTest := range tests {
		t.Run(currentTest.name, func(t *testing.T) {
			testRouter := chi.NewRouter()
			testRouter.Route("/update", func(r chi.Router) {
				r.Post("/{type}/{name}/{val}", handlerUpdate)
			})
			testServer := httptest.NewServer(testRouter)
			defer testServer.Close()

			res, err := http.Post(testServer.URL+currentTest.url, "", nil)
			assert.NoError(t, err)
			defer res.Body.Close()

			actualValue, _ := storage.getGauge("g1")

			assert.Equal(t, currentTest.expectedValue, actualValue)
			assert.Equal(t, currentTest.expectedStatusCode, res.StatusCode)
		})
	}
}

func Test_handlerGetMetric(t *testing.T) {
	storage = Metrics{
		Gauges: map[string]float64{
			"g1": 5.5,
		},
	}
	tests := []struct {
		name               string
		url                string
		expectedBody       string
		expectedStatusCode int
	}{
		{
			name:               "#1 existing metric getting",
			url:                "/value/gauge/g1",
			expectedBody:       "5.500",
			expectedStatusCode: 200,
		},
		{
			name:               "#2 unknown path",
			url:                "/values/gauge/g1",
			expectedBody:       "404 page not found\n",
			expectedStatusCode: 404,
		},
		{
			name:               "#3 unknown metric type",
			url:                "/value/gauges/g1",
			expectedBody:       "cannot get: no such metrics type <gauges>\n",
			expectedStatusCode: 501,
		},
		{
			name:               "#4 unknown metric name",
			url:                "/value/gauge/g2",
			expectedBody:       "cannot get: no such gauge <g2>\n",
			expectedStatusCode: 404,
		},
	}
	for _, currentTest := range tests {
		t.Run(currentTest.name, func(t *testing.T) {
			testRouter := chi.NewRouter()
			testRouter.Route("/value", func(r chi.Router) {
				r.Get("/{type}/{name}", handlerGetMetric)
			})
			testServer := httptest.NewServer(testRouter)
			defer testServer.Close()

			res, err := http.Get(testServer.URL + currentTest.url)
			resBody, _ := io.ReadAll(res.Body)
			assert.NoError(t, err)
			defer res.Body.Close()

			assert.Equal(t, currentTest.expectedBody, string(resBody))
			assert.Equal(t, currentTest.expectedStatusCode, res.StatusCode)
		})
	}
}

func Test_handlerGetMetricsByType(t *testing.T) {
	storage = Metrics{
		Gauges: map[string]float64{
			"g1": 5.5,
		},
		Counters: map[string]int64{
			"c1": 1,
		},
	}
	tests := []struct {
		name               string
		url                string
		expectedBody       string
		expectedStatusCode int
	}{
		{
			name:               "#1 gauges getting",
			url:                "/value/gauge",
			expectedBody:       "GAUGES LIST:\n\ng1: 5.500",
			expectedStatusCode: 200,
		},
		{
			name:               "#2 counters getting",
			url:                "/value/counter",
			expectedBody:       "COUNTERS LIST:\n\nc1: 1",
			expectedStatusCode: 200,
		},
		{
			name:               "#3 unknown path",
			url:                "/values/gauge",
			expectedBody:       "404 page not found\n",
			expectedStatusCode: 404,
		},
		{
			name:               "#4 unknown metric type",
			url:                "/value/gauges",
			expectedBody:       "cannot get: no such metrics type <gauges>\n",
			expectedStatusCode: 404,
		},
	}
	for _, currentTest := range tests {
		t.Run(currentTest.name, func(t *testing.T) {
			testRouter := chi.NewRouter()
			testRouter.Route("/value", func(r chi.Router) {
				r.Get("/{type}", handlerGetMetricsByType)
			})
			testServer := httptest.NewServer(testRouter)
			defer testServer.Close()

			res, err := http.Get(testServer.URL + currentTest.url)
			resBody, _ := io.ReadAll(res.Body)
			assert.NoError(t, err)
			defer res.Body.Close()

			assert.Equal(t, currentTest.expectedBody, string(resBody))
			assert.Equal(t, currentTest.expectedStatusCode, res.StatusCode)
		})
	}
}
