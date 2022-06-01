package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"reflect"
	"runtime"
	"strconv"
	"sync"
	"syscall"
	"time"
)

var pollInterval = 2 * time.Second
var reportInterval = 10 * time.Second
var contentType = "text/plain"
var srvAddr = "http://127.0.0.1:8080"

func main() {
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	pollTimer := time.NewTicker(pollInterval)
	reportTimer := time.NewTicker(reportInterval)
	for {
		select {
		case <-pollTimer.C:
			poll()
		case <-reportTimer.C:
			report()
		case <-signalCh:
			fmt.Println("EXIT")
			os.Exit(1)
		}
	}
}

var runtimeGauges = [...]string{
	"Alloc",
	"BuckHashSys",
	"Frees",
	"GCCPUFraction",
	"GCSys",
	"HeapAlloc",
	"HeapIdle",
	"HeapInuse",
	"HeapObjects",
	"HeapReleased",
	"HeapSys",
	"LastGC",
	"Lookups",
	"MCacheInuse",
	"MCacheSys",
	"MSpanInuse",
	"MSpanSys",
	"Mallocs",
	"NextGC",
	"NumForcedGC",
	"NumGC",
	"OtherSys",
	"PauseTotalNs",
	"StackInuse",
	"StackSys",
	"Sys",
	"TotalAlloc",
}

var customGauges = [...]string{
	"RandomValue",
}

var counters = [...]string{
	"PollCounter",
}

var metrics = struct {
	sync.RWMutex
	gauges   map[string]float64
	counters map[string]int64
}{
	gauges:   map[string]float64{},
	counters: map[string]int64{},
}

func getGauge(name string) (float64, error) {
	metrics.RLock()
	defer metrics.RUnlock()
	if val, ok := metrics.gauges[name]; ok {
		return val, nil
	}
	return 0, fmt.Errorf("cannot get: no such gauge <%v>", name)
}

func getCounter(name string) (int64, error) {
	metrics.RLock()
	defer metrics.RUnlock()
	if val, ok := metrics.counters[name]; ok {
		return val, nil
	}
	return 0, fmt.Errorf("cannot get: no such counter <%v>", name)
}

func updGaugeByRuntimeValue(name string) bool {
	metrics.Lock()
	defer metrics.Unlock()
	m := &runtime.MemStats{}
	runtime.ReadMemStats(m)
	tmp := reflect.Indirect(reflect.ValueOf(m)).FieldByName(name).Convert(reflect.TypeOf(metrics.gauges[name])).Float()
	if metrics.gauges[name] == tmp {
		return false
	}
	metrics.gauges[name] = tmp
	return true
}

func updGaugeByRandomValue(name string) {
	metrics.Lock()
	defer metrics.Unlock()
	metrics.gauges[name] = 100 * rand.Float64()
}

func updCounter(name string) {
	metrics.Lock()
	defer metrics.Unlock()
	metrics.counters[name] += 1
}

func resetCounter(name string) {
	metrics.Lock()
	defer metrics.Unlock()
	metrics.counters[name] = 0
}

func sendGauge(srvAddr, contentType, metricName string) error {
	val, err := getGauge(metricName)
	if err != nil {
		return err
	}
	err = postReq(srvAddr+"/update/gauge/"+metricName+"/"+strconv.FormatFloat(val, 'f', 3, 64), contentType)
	if err != nil {
		return err
	}
	return nil
}

func sendCounter(srvAddr, contentType, metricName string) error {
	val, err := getCounter(metricName)
	if err != nil {
		return err
	}
	err = postReq(srvAddr+"/update/counter/"+metricName+"/"+strconv.FormatInt(val, 10), contentType)
	if err != nil {
		return err
	}
	return nil
}

func postReq(addr, cont string) error {
	res, err := http.Post(addr, cont, nil)
	if err != nil {
		fmt.Printf("err: %v\n", err)
	}
	defer res.Body.Close()
	fmt.Println(res.Status, res.Request.URL)
	return nil
}

func poll() {
	resetCounter(counters[0])
	for i := range runtimeGauges {
		polled := updGaugeByRuntimeValue(runtimeGauges[i])
		if polled {
			updCounter(counters[0])
		}
	}
	updGaugeByRandomValue(customGauges[0])
}

func report() {
	go sendCounter(srvAddr, contentType, counters[0])
	go sendGauge(srvAddr, contentType, customGauges[0])
	for i := range runtimeGauges {
		go sendGauge(srvAddr, contentType, runtimeGauges[i])
	}
}
