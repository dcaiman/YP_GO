//
package server

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"sync"
	"text/template"

	"github.com/go-chi/chi/v5"
)

var srvAddr = "127.0.0.1:8080"
var storage Metrics

func RunServer() {
	storage = Metrics{
		Gauges:   map[string]float64{},
		Counters: map[string]int64{},
	}
	mainRouter := chi.NewRouter()
	mainRouter.Route("/", func(r chi.Router) {
		r.Get("/", handlerGetAll)
	})
	mainRouter.Route("/value", func(r chi.Router) {
		r.Get("/{type}", handlerGetMetricsByType)
		r.Get("/{type}/{name}", handlerGetMetric)
	})
	mainRouter.Route("/update", func(r chi.Router) {
		r.Post("/{type}/{name}/{val}", handlerUpdate)
	})

	http.ListenAndServe(srvAddr, mainRouter)
}

const (
	Gauge   = "gauge"
	Counter = "counter"
)

type Metrics struct {
	sync.RWMutex
	Gauges   map[string]float64
	Counters map[string]int64
}

func (m *Metrics) updateGauge(name string, val float64) {
	m.Lock()
	defer m.Unlock()
	m.Gauges[name] = val
}

func (m *Metrics) updateCounter(name string, val int64) {
	m.Lock()
	defer m.Unlock()
	m.Counters[name] += val
}

func (m *Metrics) getGauges() []string {
	arr := []string{}
	m.RLock()
	defer m.RUnlock()
	for k, v := range m.Gauges {
		arr = append(arr, k+": "+strconv.FormatFloat(v, 'f', 3, 64))
	}
	return arr
}

func (m *Metrics) getCounters() []string {
	arr := []string{}
	m.RLock()
	defer m.RUnlock()
	for k, v := range m.Counters {
		arr = append(arr, k+": "+strconv.FormatInt(v, 10))
	}
	return arr
}

func (m *Metrics) getGauge(name string) (float64, error) {
	m.RLock()
	defer m.RUnlock()
	if val, ok := m.Gauges[name]; ok {
		return val, nil
	}
	err := errors.New("cannot get: no such gauge <" + name + ">")
	log.Println(err.Error())
	return 0, err
}

func (m *Metrics) getCounter(name string) (int64, error) {
	m.RLock()
	defer m.RUnlock()
	if val, ok := m.Counters[name]; ok {
		return val, nil
	}
	err := errors.New("cannot get: no such counter <" + name + ">")
	log.Println(err.Error())
	return 0, err
}

func handlerGetAll(w http.ResponseWriter, r *http.Request) {
	t, _ := template.New("").Parse("GAUGES LIST:\n{{range $v := .Gauges}}\n{{$v}}{{end}}\n\nCOUNTERS LIST:\n{{range $v := .Counters}}\n{{$v}}{{end}}")
	t.Execute(w, struct {
		Gauges, Counters []string
	}{
		Gauges:   storage.getGauges(),
		Counters: storage.getCounters(),
	})
}

func handlerUpdate(w http.ResponseWriter, r *http.Request) {
	metricType := chi.URLParam(r, "type")
	metricName := chi.URLParam(r, "name")
	metricVal := chi.URLParam(r, "val")
	switch metricType {
	case Gauge:
		val, err := strconv.ParseFloat(metricVal, 64)
		if err != nil {
			log.Println(err.Error())
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		storage.updateGauge(metricName, val)
	case Counter:
		val, err := strconv.ParseInt(metricVal, 10, 64)
		if err != nil {
			log.Println(err.Error())
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		storage.updateCounter(metricName, val)
	default:
		err := errors.New("cannot update: no such metric type <" + metricType + ">")
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusNotImplemented)
		return
	}
}

func handlerGetMetric(w http.ResponseWriter, r *http.Request) {
	metricType := chi.URLParam(r, "type")
	metricName := chi.URLParam(r, "name")
	switch metricType {
	case Gauge:
		if metricVal, err := storage.getGauge(metricName); err == nil {
			w.Write([]byte(strconv.FormatFloat(metricVal, 'f', 3, 64)))
		} else {
			log.Println(err.Error())
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
	case Counter:
		if metricVal, err := storage.getCounter(metricName); err == nil {
			w.Write([]byte(strconv.FormatInt(metricVal, 10)))
		} else {
			log.Println(err.Error())
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
	default:
		err := errors.New("cannot get: no such metrics type <" + metricType + ">")
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusNotImplemented)
		return
	}
}

func handlerGetMetricsByType(w http.ResponseWriter, r *http.Request) {
	metricType := chi.URLParam(r, "type")
	switch metricType {
	case Gauge:
		t, _ := template.New("").Parse("GAUGES LIST:\n{{range $v := .}}\n{{$v}}{{end}}")
		t.Execute(w, storage.getGauges())
	case Counter:
		t, _ := template.New("").Parse("COUNTERS LIST:\n{{range $v := .}}\n{{$v}}{{end}}")
		t.Execute(w, storage.getCounters())
	default:
		err := errors.New("cannot get: no such metrics type <" + metricType + ">")
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
}
