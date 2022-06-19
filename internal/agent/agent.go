package agent

import (
	"YP_GO_devops/internal/metrics"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/caarlos0/env"
)

var storage metrics.Metrics
var cfg EnvConfig

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

type EnvConfig struct {
	PollInterval   time.Duration `env:"POLL_INTERVAL"`
	ReportInterval time.Duration `env:"REPORT_INTERVAL"`
	SrvAddr        string        `env:"ADDRESS"`
}

func RunAgent() {
	storage = metrics.Metrics{
		Gauges:   map[string]float64{},
		Counters: map[string]int64{},
	}
	cfg = EnvConfig{
		PollInterval:   3 * time.Second,
		ReportInterval: 7 * time.Second,
		SrvAddr:        "127.0.0.1:8080",
	}
	if err := env.Parse(&cfg); err != nil {
		log.Println(err.Error())
	}
	fmt.Println(cfg)
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	pollTimer := time.NewTicker(cfg.PollInterval)
	reportTimer := time.NewTicker(cfg.ReportInterval)
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
		storage.UpdateGaugeByRuntimeValue(runtimeGauges[i])
	}
	storage.UpdateGaugeByRandomValue(customGauges[0])
	storage.IncreaseCounter(counters[0], 1)
}

func report() {
	go storage.SendMetric(cfg.SrvAddr, metrics.JsonCT, counters[0], metrics.Counter)
	go storage.SendMetric(cfg.SrvAddr, metrics.JsonCT, customGauges[0], metrics.Gauge)
	for i := range runtimeGauges {
		go storage.SendMetric(cfg.SrvAddr, metrics.JsonCT, runtimeGauges[i], metrics.Gauge)
	}
}
