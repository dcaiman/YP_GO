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

type AgentEnv struct {
	RuntimeGauges []string
	CustomGauges  []string
	Counters      []string
	Storage       metric.MStorage
	Cfg           EnvConfig
}

func RunAgent(agn *AgentEnv) {
	log.Println("AGENT CONFIG: ", agn.Cfg)

	fileStorage := internalstorage.New("")
	agn.Storage = fileStorage

	go agn.poll()
	go agn.report()

	syscallCh := make(chan os.Signal, 1)
	signal.Notify(syscallCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	<-syscallCh
	log.Println("EXIT")
	os.Exit(0)
}

func (agn *AgentEnv) GetExternalConfig() error {
	if agn.Cfg.ArgConfig {
		flag.StringVar(&agn.Cfg.SrvAddr, "a", agn.Cfg.SrvAddr, "server address")
		flag.DurationVar(&agn.Cfg.ReportInterval, "r", agn.Cfg.ReportInterval, "report interval")
		flag.DurationVar(&agn.Cfg.PollInterval, "p", agn.Cfg.PollInterval, "poll interval")
		flag.StringVar(&agn.Cfg.HashKey, "k", agn.Cfg.HashKey, "hash key")
		flag.Parse()
	}
	if agn.Cfg.EnvConfig {
		if err := env.Parse(&agn.Cfg); err != nil {
			return clog.ToLog(clog.FuncName(), err)
		}
	}
	return nil
}
