package agent

import (
	"bytes"
	"errors"
	"flag"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"reflect"
	"runtime"
	"strconv"
	"syscall"
	"time"

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
	"RandomValue7",
}

var counters = [...]string{
	"PollCount7",
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

	prepareStorage(agn)

	for {
		select {
		case <-pollTimer.C:
			poll(agn)
		case <-reportTimer.C:
			report(agn)
		case <-signalCh:
			log.Println("EXIT")
			os.Exit(0)
		}
	}
}

func prepareStorage(agn *AgentConfig) {
	for i := range runtimeGauges {
		m := metric.Metric{
			ID:    runtimeGauges[i],
			MType: Gauge,
		}
		m.UpdateHash(agn.Cfg.HashKey)
		agn.Storage.NewMetric(m)
	}
	for i := range customGauges {
		m := metric.Metric{
			ID:    customGauges[i],
			MType: Gauge,
		}
		m.UpdateHash(agn.Cfg.HashKey)
		agn.Storage.NewMetric(m)
	}
	for i := range counters {
		m := metric.Metric{
			ID:    counters[i],
			MType: Counter,
		}
		m.UpdateHash(agn.Cfg.HashKey)
		agn.Storage.NewMetric(m)
	}
}

func poll(agn *AgentConfig) {
	for i := range runtimeGauges {
		if err := agn.Storage.UpdateValue(runtimeGauges[i], getRuntimeMetric(runtimeGauges[i])); err != nil {
			log.Println(err.Error())
		}
	}
	for i := range customGauges {
		if err := agn.Storage.UpdateValue(customGauges[i], 100*rand.Float64()); err != nil {
			log.Println(err.Error())
		}
	}
	for i := range counters {
		if err := agn.Storage.IncreaseDelta(counters[i]); err != nil {
			log.Println(err.Error())
		}
	}
}

func report(agn *AgentConfig) {
	for i := range runtimeGauges {
		go sendMetric(runtimeGauges[i], agn)
	}
	for i := range customGauges {
		go sendMetric(customGauges[i], agn)
	}
	for i := range counters {
		go sendMetric(counters[i], agn)
	}
}

func getRuntimeMetric(name string) float64 {
	mem := &runtime.MemStats{}
	runtime.ReadMemStats(mem)
	return reflect.Indirect(reflect.ValueOf(mem)).FieldByName(name).Convert(reflect.TypeOf(0.0)).Float()
}

func sendMetric(name string, agn *AgentConfig) error {
	var url, val string
	var body []byte

	m, err := agn.Storage.GetMetric(name)
	if err != nil {
		log.Println(err.Error())
		return err
	}

	switch agn.Cfg.CType {
	case TextPlainCT:
		switch m.MType {
		case Gauge:
			val = strconv.FormatFloat(*m.Value, 'f', 3, 64)
		case Counter:
			val = strconv.FormatInt(*m.Delta, 10)
		default:
			err := errors.New("cannot send: unsupported metric type <" + m.MType + ">")
			log.Println(err)
			return err
		}
		url = agn.Cfg.SrvAddr + "/update/" + m.MType + "/" + m.ID + "/" + val
		body = nil
	case JSONCT:
		tmpBody, err := m.GetJSON()
		if err != nil {
			log.Println(err.Error())
			return err
		}
		url = agn.Cfg.SrvAddr + "/update/"
		body = tmpBody
	default:
		err := errors.New("cannot send: unsupported content type <" + agn.Cfg.CType + ">")
		log.Println(err)
		return err
	}
	res, err := customPostRequest(HTTPStr+url, agn.Cfg.CType, m.Hash, bytes.NewBuffer(body))
	if err != nil {
		log.Println(err.Error())
		return err
	}
	defer res.Body.Close()
	if m.MType == Counter {
		agn.Storage.ResetDelta(name)
	}
	log.Println(res.Status, res.Request.URL)
	return nil
}

func customPostRequest(url, contentType, hash string, body io.Reader) (resp *http.Response, err error) {
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}
	if hash != "" {
		req.Header.Set("Hash", hash)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	return http.DefaultClient.Do(req)
}

/*
func getBatch(metricNames []string, agn *AgentConfig) ([]metric.Metric, error) {
	batch := []metric.Metric{}
	for i := range metricNames {
		m, err := agn.Storage.GetMetric(metricNames[i])
		if err != nil {
			return nil, err
		}
		batch = append(batch, m)
	}
	return batch, nil
}
*/
