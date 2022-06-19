package server

import (
	"YP_GO_devops/internal/metrics"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"
	"text/template"

	"github.com/go-chi/chi/v5"
)

func handlerUpdateJSON(w http.ResponseWriter, r *http.Request) {
	content, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	mj := metrics.MetricJSON{}
	if err := json.Unmarshal(content, &mj); err != nil {
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	switch mj.MType {
	case metrics.Gauge:
		storage.UpdateGaugeByValue(mj.ID, *mj.Value)
	case metrics.Counter:
		storage.IncreaseCounter(mj.ID, *mj.Delta)
	default:
		err := errors.New("cannot update: no such metric type <" + mj.ID + ">")
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusNotImplemented)
		return
	}
}

func handlerUpdateDirect(w http.ResponseWriter, r *http.Request) {
	metricType := chi.URLParam(r, "type")
	metricName := chi.URLParam(r, "name")
	metricVal := chi.URLParam(r, "val")
	switch metricType {
	case metrics.Gauge:
		val, err := strconv.ParseFloat(metricVal, 64)
		if err != nil {
			log.Println(err.Error())
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		storage.UpdateGaugeByValue(metricName, val)
	case metrics.Counter:
		val, err := strconv.ParseInt(metricVal, 10, 64)
		if err != nil {
			log.Println(err.Error())
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		storage.IncreaseCounter(metricName, val)
	default:
		err := errors.New("cannot update: no such metric type <" + metricType + ">")
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusNotImplemented)
		return
	}
}

func handlerGetAll(w http.ResponseWriter, r *http.Request) {
	t, _ := template.New("").Parse("GAUGES LIST:\n{{range $v := .Gauges}}\n{{$v}}{{end}}\n\nCOUNTERS LIST:\n{{range $v := .Counters}}\n{{$v}}{{end}}")
	t.Execute(w, struct {
		Gauges, Counters []string
	}{
		Gauges:   storage.GetGauges(),
		Counters: storage.GetCounters(),
	})
}

func handlerGetMetricJSON(w http.ResponseWriter, r *http.Request) {
	content, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	mj := metrics.MetricJSON{}
	if err := json.Unmarshal(content, &mj); err != nil {
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	switch mj.MType {
	case metrics.Gauge:
		val, err := storage.GetGauge(mj.ID)
		if err != nil {
			log.Println(err.Error())
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		mj.Value = &val
	case metrics.Counter:
		val, err := storage.GetCounter(mj.ID)
		if err != nil {
			log.Println(err.Error())
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		mj.Delta = &val
	default:
		err := errors.New("cannot get: no such metrics type <" + mj.MType + ">")
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusNotImplemented)
		return
	}
	mjRes, err := json.Marshal(mj)
	if err != nil {
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", metrics.JSONCT)
	w.Write(mjRes)
}

func handlerGetMetric(w http.ResponseWriter, r *http.Request) {
	metricType := chi.URLParam(r, "type")
	metricName := chi.URLParam(r, "name")
	switch metricType {
	case metrics.Gauge:
		if metricVal, err := storage.GetGauge(metricName); err == nil {
			w.Write([]byte(strconv.FormatFloat(metricVal, 'f', 3, 64)))
		} else {
			log.Println(err.Error())
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
	case metrics.Counter:
		if metricVal, err := storage.GetCounter(metricName); err == nil {
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
	case metrics.Gauge:
		t, _ := template.New("").Parse("GAUGES LIST:\n{{range $v := .}}\n{{$v}}{{end}}")
		t.Execute(w, storage.GetGauges())
	case metrics.Counter:
		t, _ := template.New("").Parse("COUNTERS LIST:\n{{range $v := .}}\n{{$v}}{{end}}")
		t.Execute(w, storage.GetCounters())
	default:
		err := errors.New("cannot get: no such metrics type <" + metricType + ">")
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
}
