package main

import (
	"net/http"
	"strconv"
	"strings"
	"sync"
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

func main() {
	http.HandleFunc("/", hdlrUpdate)
	http.ListenAndServe(srvAddr, nil)
}

func hdlrUpdate(ww http.ResponseWriter, rr *http.Request) {
	s := strings.Split(rr.URL.String(), "/")
	switch s[2] {
	case "gauge":
		val, _ := strconv.ParseFloat(s[4], 64)
		updGauge(s[3], val)
	case "counter":
		val, _ := strconv.ParseInt(s[4], 10, 64)
		updCounter(s[3], val)
	default:
		http.Error(ww, "can't handle this metric type", http.StatusBadRequest)
		return
	}
}
