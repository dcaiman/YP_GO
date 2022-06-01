package server

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
)

func hdlrGetAll(ww http.ResponseWriter, rr *http.Request) {
	ww.Write([]byte("METRICS LIST\n\n" + strings.Join(getGauges(), "\n") + "\n\n" + strings.Join(getCounters(), "\n")))
}

func hdlrUpdate(ww http.ResponseWriter, rr *http.Request) {
	metricType := chi.URLParam(rr, "type")
	metricName := chi.URLParam(rr, "name")
	metricVal := chi.URLParam(rr, "val")
	switch metricType {
	case "gauge":
		val, err := strconv.ParseFloat(metricVal, 64)
		if err != nil {
			http.Error(ww, err.Error(), http.StatusBadRequest)
			return
		}
		updGauge(metricName, val)
	case "counter":
		val, err := strconv.ParseInt(metricVal, 10, 64)
		if err != nil {
			http.Error(ww, err.Error(), http.StatusBadRequest)
		}
		updCounter(metricName, val)
	default:
		http.Error(ww, "cannot update: no such metric type <"+metricType+">", http.StatusNotImplemented)
		return
	}
}

func hdlrGetMetric(ww http.ResponseWriter, rr *http.Request) {
	metricType := chi.URLParam(rr, "type")
	metricName := chi.URLParam(rr, "name")
	switch metricType {
	case "gauge":
		if metricVal, err := getGauge(metricName); err == nil {
			ww.Write([]byte(strconv.FormatFloat(metricVal, 'f', 3, 64)))
		} else {
			http.Error(ww, err.Error(), http.StatusNotFound)
			return
		}
	case "counter":
		if metricVal, err := getCounter(metricName); err == nil {
			ww.Write([]byte(strconv.FormatInt(metricVal, 10)))
		} else {
			http.Error(ww, err.Error(), http.StatusNotFound)
			return
		}
	default:
		http.Error(ww, "cannot get: no such metrics type <"+metricType+">", http.StatusNotImplemented)
		return
	}
}

func hdlrGetMetricsByType(ww http.ResponseWriter, rr *http.Request) {
	metricType := chi.URLParam(rr, "type")
	switch metricType {
	case "gauge":
		ww.Write([]byte("GAUGES LIST:\n\n" + strings.Join(getGauges(), "\n")))
	case "counter":
		ww.Write([]byte("COUNTERS LIST:\n\n" + strings.Join(getCounters(), "\n")))
	default:
		http.Error(ww, "cannot get: no such metrics type <"+metricType+">", http.StatusNotFound)
		return
	}
}
