package main

import (
	"net/http"
	"strconv"
	"strings"
	"sync"
)

var srvAddr = "127.0.0.1:8080"

func main() {
	http.HandleFunc("/", hdlrUpdate)
	http.ListenAndServe(srvAddr, nil)
}

var Metrics = struct {
	sync.RWMutex
	Gauges   map[string]float64
	Counters map[string]int64
}{
	Gauges:   map[string]float64{},
	Counters: map[string]int64{},
}

func updGauge(name string, val float64) {
	Metrics.Lock()
	defer Metrics.Unlock()
	Metrics.Gauges[name] = val
}

func updCounter(name string, val int64) {
	Metrics.Lock()
	defer Metrics.Unlock()
	Metrics.Counters[name] += val
}

func hdlrUpdate(ww http.ResponseWriter, rr *http.Request) {
	s := strings.Split(rr.URL.String(), "/")
	metricType := s[2]
	metricName := s[3]
	metricVal := s[4]
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
