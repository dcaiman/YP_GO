package agent

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var pollInterval = 2 * time.Second
var reportInterval = 10 * time.Second
var srvAddr = "http://127.0.0.1:8080"
var storage Metrics

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
	"PollCount",
}

func RunAgent() {
	storage = Metrics{
		Gauges:   map[string]float64{},
		Counters: map[string]int64{},
	}

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
			log.Println("EXIT")
			os.Exit(0)
		}
	}
}

func poll() {
	for i := range runtimeGauges {
		storage.updateGaugeByRuntimeValue(runtimeGauges[i])
	}
	storage.updateGaugeByRandomValue(customGauges[0])
	storage.updateCounter(counters[0], 1)
	storage.updateMetricFromServer(srvAddr, "dummy", Gauge)
}

func report() {
	go storage.sendCounter(srvAddr, jsonCT, counters[0])
	go storage.sendGauge(srvAddr, jsonCT, customGauges[0])
	for i := range runtimeGauges {
		go storage.sendGauge(srvAddr, jsonCT, runtimeGauges[i])
	}
}
