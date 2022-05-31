package main

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"sync"

	"github.com/go-chi/chi/v5"
)

var srvAddr = "127.0.0.1:8080"

var Gauges = struct {
	sync.RWMutex
	m map[string]float64
}{m: make(map[string]float64)}

var Counters = struct {
	sync.RWMutex
	m map[string]int64
}{m: make(map[string]int64)}

func updGauge(name string, val float64) {
	Gauges.Lock()
	Gauges.m[name] = val
	Gauges.Unlock()
}

func updCounter(name string, val int64) {
	Counters.Lock()
	Counters.m[name] += val
	Counters.Unlock()
}

func getGauges() []string {
	arr := []string{}
	Gauges.RLock()
	for k, v := range Gauges.m {
		arr = append(arr, k+": "+strconv.FormatFloat(v, 'f', 3, 64))
	}
	Gauges.RUnlock()
	return arr
}

func getCounters() []string {
	arr := []string{}
	Counters.RLock()
	for k, v := range Counters.m {
		arr = append(arr, k+": "+strconv.FormatInt(v, 10))
	}
	Counters.RUnlock()
	return arr
}

func getGauge(name string) (float64, error) {
	Gauges.RLock()
	if val, ok := Gauges.m[name]; ok {
		return val, nil
	}
	Gauges.RUnlock()
	return 0, fmt.Errorf("no such metric name <%v>", name)
}

func getCounter(name string) (int64, error) {
	Counters.RLock()
	if val, ok := Counters.m[name]; ok {
		return val, nil
	}
	Counters.RUnlock()
	return 0, fmt.Errorf("no such metric name <%v>", name)
}

func main() {
	mainRouter := chi.NewRouter()
	mainRouter.Route("/", func(r chi.Router) {
		r.Get("/", hdlrGetAll)
	})
	mainRouter.Route("/value", func(r chi.Router) {
		r.Get("/{type}", hdlrGetMetricsByType)
		r.Get("/{type}/{name}", hdlrGetMetric)
	})
	mainRouter.Route("/update", func(r chi.Router) {
		r.Post("/{type}/{name}/{val}", hdlrUpdate)
	})
	http.ListenAndServe(srvAddr, mainRouter)
}

func hdlrGetAll(ww http.ResponseWriter, rr *http.Request) {
	ww.Write([]byte("METRICS LIST\n\n" + strings.Join(getGauges(), "\n") + "\n\n" + strings.Join(getCounters(), "\n")))
}

func hdlrUpdate(ww http.ResponseWriter, rr *http.Request) {
	metricType := chi.URLParam(rr, "type")
	metricName := chi.URLParam(rr, "name")
	metricVal := chi.URLParam(rr, "val")
	switch metricType {
	case "gauge":
		val, _ := strconv.ParseFloat(metricVal, 64)
		updGauge(metricName, val)
	case "counter":
		val, _ := strconv.ParseInt(metricVal, 10, 64)
		updCounter(metricName, val)
	default:
		http.Error(ww, "can't handle this metric type", http.StatusBadRequest)
		return
	}
}

func hdlrGetMetric(ww http.ResponseWriter, rr *http.Request) {
	metricType := chi.URLParam(rr, "type")
	metricName := chi.URLParam(rr, "name")
	switch metricType {
	case "gauge":
		if metricVal, err := getGauge(metricName); err == nil {
			ww.Write([]byte(metricName + ": " + strconv.FormatFloat(metricVal, 'f', 3, 64)))
		} else {
			http.Error(ww, err.Error(), http.StatusNotFound)
			return
		}
	case "counter":
		if metricVal, err := getCounter(metricName); err == nil {
			ww.Write([]byte(metricName + ": " + strconv.FormatInt(metricVal, 10)))
		} else {
			http.Error(ww, err.Error(), http.StatusNotFound)
			return
		}
	default:
		http.Error(ww, "no such metrics type <"+metricType+">", http.StatusNotFound)
		return
	}
}

func hdlrGetMetricsByType(ww http.ResponseWriter, rr *http.Request) {
	metricType := chi.URLParam(rr, "type")
	switch metricType {
	case "gauge":
		list := getGauges()
		if list != nil {
			ww.Write([]byte("GAUGES LIST:\n\n" + strings.Join(list, "\n")))
		} else {
			http.Error(ww, "no metrics of type <"+metricType+">", http.StatusNotFound)
			return
		}
	case "counter":
		list := getCounters()
		if list != nil {
			ww.Write([]byte("COUNTERS LIST:\n\n" + strings.Join(list, "\n")))
		} else {
			http.Error(ww, "no metrics of type <"+metricType+">", http.StatusNotFound)
			return
		}
	default:
		http.Error(ww, "no such metrics type <"+metricType+">", http.StatusNotFound)
	}
}
