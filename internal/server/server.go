//
package server

import (
	"flag"
	"log"
	"net/http"
	"time"

	"github.com/dcaiman/YP_GO/internal/metrics"

	"github.com/caarlos0/env"
	"github.com/go-chi/chi/v5"
)

type EnvConfig struct {
	SrvAddr       string        `env:"ADDRESS"`
	StoreInterval time.Duration `env:"STORE_INTERVAL"`
	StoreFile     string        `env:"STORE_FILE"`
	InitDownload  bool          `env:"RESTORE"`
	HashKey       string        `env:"KEY"`
	EnvConfig     bool
	ArgConfig     bool
	SyncUpload    bool
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
		StoreInterval: 0,
		StoreFile:     "/tmp/metricStorage.json",
		HashKey:       "key",
		InitDownload:  true,
		ArgConfig:     true,
		EnvConfig:     true,
	}
	if cfg.ArgConfig {
		flag.BoolVar(&cfg.InitDownload, "r", cfg.InitDownload, "initial download flag")
		flag.StringVar(&cfg.StoreFile, "f", cfg.StoreFile, "storage file destination")
		flag.StringVar(&cfg.SrvAddr, "a", cfg.SrvAddr, "server address")
		flag.DurationVar(&cfg.StoreInterval, "i", cfg.StoreInterval, "store interval")
		flag.StringVar(&cfg.HashKey, "k", cfg.HashKey, "hash key")
		flag.Parse()
	}
	if cfg.EnvConfig {
		if err := env.Parse(&cfg); err != nil {
			log.Println(err.Error())
		}
	}
	if cfg.InitDownload && cfg.StoreFile != "" {
		err := storage.DownloadStorage(cfg.StoreFile)
		if err != nil {
			log.Println(err.Error())
		}
	}
	if cfg.StoreInterval != 0 {
		go func() {
			uploadTimer := time.NewTicker(cfg.StoreInterval)
			for {
				<-uploadTimer.C
				err := storage.UploadStorage(cfg.StoreFile)
				if err != nil {
					log.Println(err.Error())
				}
			}
		}()
	} else {
		cfg.SyncUpload = true
	}
	log.Println("SERVER CONFIG: ", cfg)

	storage.EncryptingKey = cfg.HashKey

	mainRouter := chi.NewRouter()
	mainRouter.Route("/", func(r chi.Router) {
		r.Get("/", Compresser(handlerGetAll))
	})
	mainRouter.Route("/value", func(r chi.Router) {
		r.Post("/", Compresser(handlerGetMetricJSON))
		r.Get("/{type}", Compresser(handlerGetMetricsByType))
		r.Get("/{type}/{name}", Compresser(handlerGetMetric))
	})
	mainRouter.Route("/update", func(r chi.Router) {
		r.Post("/", Compresser(handlerUpdateJSON))
		r.Post("/{type}/{name}/{val}", Compresser(handlerUpdateDirect))
	})
	log.Println(http.ListenAndServe(cfg.SrvAddr, mainRouter))
}
