//
package server

import (
	"YP_GO_devops/internal/metrics"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/caarlos0/env"
	"github.com/go-chi/chi/v5"
)

type EnvConfig struct {
	SrvAddr       string        `env:"ADDRESS"`
	StoreInterval time.Duration `env:"STORE_INTERVAL"`
	StoreFile     string        `env:"STORE_FILE"`
	InitDownload  bool          `env:"RESTORE"`
	EnvConfig     bool
}

var cfg EnvConfig
var storage metrics.Metrics

func RunServer() {
	storage = metrics.Metrics{
		Gauges:   map[string]float64{},
		Counters: map[string]int64{},
	}
	cfg = EnvConfig{
		SrvAddr:       "127.0.0.1:8080",
		StoreInterval: 5 * time.Second,
		StoreFile:     "/tmp/devops-metrics-db",
		InitDownload:  true,
		EnvConfig:     true,
	}
	fmt.Println(cfg)
	if cfg.EnvConfig {
		if err := env.Parse(&cfg); err != nil {
			log.Println("env.Parse", err.Error())
		}
	}
	fmt.Println(cfg)
	if cfg.InitDownload && cfg.StoreFile != "" {
		err := storage.DownloadStorage(cfg.StoreFile)
		if err != nil {
			log.Println("storage.DownloadStorage", err.Error())
		}
	}
	if cfg.StoreInterval != 0 {
		go func() {
			uploadTimer := time.NewTicker(cfg.StoreInterval)
			for {
				<-uploadTimer.C
				err := storage.UploadStorage(cfg.StoreFile)
				if err != nil {
					log.Println("storage.UploadStorage", err.Error())
				}
			}
		}()
	}

	mainRouter := chi.NewRouter()
	mainRouter.Route("/", func(r chi.Router) {
		r.Get("/", handlerGetAll)
	})
	mainRouter.Route("/value", func(r chi.Router) {
		r.Post("/", handlerGetMetricJSON)
		r.Get("/{type}", handlerGetMetricsByType)
		r.Get("/{type}/{name}", handlerGetMetric)
	})
	mainRouter.Route("/update", func(r chi.Router) {
		r.Post("/", handlerUpdateJSON)
		r.Post("/{type}/{name}/{val}", handlerUpdateDirect)
	})
	log.Println(http.ListenAndServe(cfg.SrvAddr, mainRouter))
}
