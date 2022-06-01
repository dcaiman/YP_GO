package main

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"sync"

	"github.com/go-chi/chi/v5"
)

var Metrics = struct {
	sync.RWMutex
	Gauges   map[string]float64
	Counters map[string]int64
}{
	Gauges:   map[string]float64{},
	Counters: map[string]int64{},
}

/*
{
	Gauges: map[string]float64{
		"Alloc":         0,
		"BuckHashSys":   0,
		"Frees":         0,
		"GCCPUFraction": 0,
		"GCSys":         0,
		"HeapAlloc":     0,
		"HeapIdle":      0,
		"HeapInuse":     0,
		"HeapObjects":   0,
		"HeapReleased":  0,
		"HeapSys":       0,
		"LastGC":        0,
		"Lookups":       0,
		"MCacheInuse":   0,
		"MCacheSys":     0,
		"MSpanInuse":    0,
		"MSpanSys":      0,
		"Mallocs":       0,
		"NextGC":        0,
		"NumForcedGC":   0,
		"NumGC":         0,
		"OtherSys":      0,
		"PauseTotalNs":  0,
		"StackInuse":    0,
		"StackSys":      0,
		"Sys":           0,
		"TotalAlloc":    0,
		"RandomValue":   0,
	},
	Counters: map[string]int64{
		"PollCounter": 0,
	},
}

func updGauge(name string, val float64) error {
	Metrics.Lock()
	defer Metrics.Unlock()
	if _, ok := Metrics.Gauges[name]; ok {
		Metrics.Gauges[name] = val
		return nil
	}
	return fmt.Errorf("cannot update: no such gauge <%v>", name)
}

func updCounter(name string, val int64) error {
	Metrics.Lock()
	defer Metrics.Unlock()
	if _, ok := Metrics.Counters[name]; ok {
		Metrics.Counters[name] += val
		return nil
	}
	return fmt.Errorf("cannot update: no such counter <%v>", name)
}
*/
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

func getGauges() []string {
	arr := []string{}
	Metrics.RLock()
	defer Metrics.RUnlock()
	for k, v := range Metrics.Gauges {
		arr = append(arr, k+": "+strconv.FormatFloat(v, 'f', 3, 64))
	}
	return arr
}

func getCounters() []string {
	arr := []string{}
	Metrics.RLock()
	defer Metrics.RUnlock()
	for k, v := range Metrics.Counters {
		arr = append(arr, k+": "+strconv.FormatInt(v, 10))
	}
	return arr
}

func getGauge(name string) (float64, error) {
	Metrics.RLock()
	defer Metrics.RUnlock()
	if val, ok := Metrics.Gauges[name]; ok {
		return val, nil
	}
	return 0, fmt.Errorf("cannot get: no such gauge <%v>", name)
}

func getCounter(name string) (int64, error) {
	Metrics.RLock()
	defer Metrics.RUnlock()
	if val, ok := Metrics.Counters[name]; ok {
		return val, nil
	}
	return 0, fmt.Errorf("cannot get: no such counter <%v>", name)
}

var srvAddr = "127.0.0.1:8080"

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
