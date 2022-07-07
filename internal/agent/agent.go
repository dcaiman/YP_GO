package agent

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dcaiman/YP_GO/internal/clog"
	"github.com/dcaiman/YP_GO/internal/internalstorage"
	"github.com/dcaiman/YP_GO/internal/metric"

	"github.com/caarlos0/env"
)

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

const (
	Gauge       = "gauge"
	Counter     = "counter"
	TextPlainCT = "text/plain"
	JSONCT      = "application/json"
	HTTPStr     = "http://"
)

type EnvConfig struct {
	PollInterval   time.Duration `env:"POLL_INTERVAL"`
	ReportInterval time.Duration `env:"REPORT_INTERVAL"`
	SrvAddr        string        `env:"ADDRESS"`
	HashKey        string        `env:"KEY"`

	CType string

	EnvConfig bool
	ArgConfig bool
	SendBatch bool
}

type AgentConfig struct {
	Storage metric.MStorage
	Cfg     EnvConfig
}

func RunAgent(agn *AgentConfig) {
	if agn.Cfg.ArgConfig {
		flag.StringVar(&agn.Cfg.SrvAddr, "a", agn.Cfg.SrvAddr, "server address")
		flag.DurationVar(&agn.Cfg.ReportInterval, "r", agn.Cfg.ReportInterval, "report interval")
		flag.DurationVar(&agn.Cfg.PollInterval, "p", agn.Cfg.PollInterval, "poll interval")
		flag.StringVar(&agn.Cfg.HashKey, "k", agn.Cfg.HashKey, "hash key")
		flag.Parse()
	}
	if agn.Cfg.EnvConfig {
		if err := env.Parse(&agn.Cfg); err != nil {
			log.Println(err.Error())
		}
	}
	log.Println("AGENT CONFIG: ", agn.Cfg)

	fileStorage := internalstorage.New("", agn.Cfg.HashKey)
	agn.Storage = fileStorage

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	pollTimer := time.NewTicker(agn.Cfg.PollInterval)
	reportTimer := time.NewTicker(agn.Cfg.ReportInterval)

	if err := agn.prepareStorage(); err != nil {
		log.Println(clog.ToLog(clog.FuncName(), err))
	}

	for {
		select {
		case <-pollTimer.C:
			if err := agn.poll(); err != nil {
				log.Println(clog.ToLog(clog.FuncName(), err))
			}
		case <-reportTimer.C:
			if err := agn.report(agn.Cfg.SendBatch); err != nil {
				log.Println(clog.ToLog(clog.FuncName(), err))
			}
		case <-signalCh:
			log.Println("EXIT")
			os.Exit(0)
		}
	}
}
