package server

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"text/template"

	"github.com/go-chi/chi/v5"
)

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
	switch r.Header.Get("Content-Type") {
	case textPlainCT:
		handlerUpdateText(w, r)
	case jsonCT:
		handlerUpdateJSON(w, r)
	}
}

func handlerUpdateJSON(w http.ResponseWriter, r *http.Request) {}

func handlerUpdateText(w http.ResponseWriter, r *http.Request) {
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
