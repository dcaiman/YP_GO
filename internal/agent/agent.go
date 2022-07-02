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

	"github.com/dcaiman/YP_GO/internal/metrics"

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
	EnvConfig      bool
	ArgConfig      bool
}

type AgentConfig struct {
	Storage metrics.MetricStorage
	Cfg     EnvConfig
}

func RunAgent(agn *AgentConfig) {
	agn.Storage.Init()

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

	agn.Storage.EncryptingKey = agn.Cfg.HashKey

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	pollTimer := time.NewTicker(agn.Cfg.PollInterval)
	reportTimer := time.NewTicker(agn.Cfg.ReportInterval)

	prepareStorage(&agn.Storage)

	for {
		select {
		case <-pollTimer.C:
			poll(&agn.Storage)
		case <-reportTimer.C:
			report(&agn.Storage, &agn.Cfg)
		case <-signalCh:
			log.Println("EXIT")
			os.Exit(0)
		}
	}
}

func prepareStorage(st *metrics.MetricStorage) {
	for i := range runtimeGauges {
		st.NewMetric(runtimeGauges[i], Gauge, nil, nil)
	}
	for i := range customGauges {
		st.NewMetric(customGauges[i], Gauge, nil, nil)
	}
	for i := range counters {
		st.NewMetric(counters[i], Counter, nil, nil)
	}
}

func poll(st *metrics.MetricStorage) {
	for i := range runtimeGauges {
		if err := st.UpdateValue(runtimeGauges[i], getRuntimeMetric(runtimeGauges[i])); err != nil {
			log.Println(err.Error())
		}
	}
	for i := range customGauges {
		if err := st.UpdateValue(customGauges[i], 100*rand.Float64()); err != nil {
			log.Println(err.Error())
		}
	}
	for i := range counters {
		if err := st.IncreaseDelta(counters[i]); err != nil {
			log.Println(err.Error())
		}
	}
}

func report(st *metrics.MetricStorage, cfg *EnvConfig) {
	for i := range runtimeGauges {
		go sendMetric(cfg.SrvAddr, JSONCT, runtimeGauges[i], st)
	}
	for i := range customGauges {
		go sendMetric(cfg.SrvAddr, JSONCT, customGauges[i], st)
	}
	for i := range counters {
		go sendMetric(cfg.SrvAddr, JSONCT, counters[i], st)
	}
}

func getRuntimeMetric(name string) float64 {
	mem := &runtime.MemStats{}
	runtime.ReadMemStats(mem)
	return reflect.Indirect(reflect.ValueOf(mem)).FieldByName(name).Convert(reflect.TypeOf(0.0)).Float()
}

func sendMetric(srvAddr, contentType, name string, st *metrics.MetricStorage) error {
	var url, val string
	var body []byte

	m, err := st.GetMetric(name)
	if err != nil {
		log.Println(err.Error())
		return err
	}

	switch contentType {
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
		url = srvAddr + "/update/" + m.MType + "/" + m.ID + "/" + val
		body = nil
	case JSONCT:
		tmpBody, err := m.GetJSON()
		if err != nil {
			log.Println(err.Error())
			return err
		}
		url = srvAddr + "/update/"
		body = tmpBody
	default:
		err := errors.New("cannot send: unsupported content type <" + contentType + ">")
		log.Println(err)
		return err
	}
	res, err := customPostRequest(HTTPStr+url, contentType, "", bytes.NewBuffer(body))
	if err != nil {
		log.Println(err.Error())
		return err
	}
	defer res.Body.Close()
	if m.MType == Counter {
		st.ResetDelta(name)
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
