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

type ServerConfig struct {
	Storage metrics.MetricStorage
	Cfg     EnvConfig
}

func RunServer(srv *ServerConfig) {
	srv.Storage.Init()

	if srv.Cfg.ArgConfig {
		flag.BoolVar(&srv.Cfg.InitDownload, "r", srv.Cfg.InitDownload, "initial download flag")
		flag.StringVar(&srv.Cfg.StoreFile, "f", srv.Cfg.StoreFile, "storage file destination")
		flag.StringVar(&srv.Cfg.SrvAddr, "a", srv.Cfg.SrvAddr, "server address")
		flag.DurationVar(&srv.Cfg.StoreInterval, "i", srv.Cfg.StoreInterval, "store interval")
		flag.StringVar(&srv.Cfg.HashKey, "k", srv.Cfg.HashKey, "hash key")
		flag.Parse()
	}
	if srv.Cfg.EnvConfig {
		if err := env.Parse(&srv.Cfg); err != nil {
			log.Println(err.Error())
		}
	}
	if srv.Cfg.InitDownload && srv.Cfg.StoreFile != "" {
		err := srv.Storage.DownloadStorage(srv.Cfg.StoreFile)
		if err != nil {
			log.Println(err.Error())
		}
	}
	if srv.Cfg.StoreInterval != 0 {
		go func() {
			uploadTimer := time.NewTicker(srv.Cfg.StoreInterval)
			for {
				<-uploadTimer.C
				err := srv.Storage.UploadStorage(srv.Cfg.StoreFile)
				if err != nil {
					log.Println(err.Error())
				}
			}
		}()
	} else {
		srv.Cfg.SyncUpload = true
	}
	log.Println("SERVER CONFIG: ", srv.Cfg)

	srv.Storage.EncryptingKey = srv.Cfg.HashKey

	mainRouter := chi.NewRouter()
	mainRouter.Route("/", func(r chi.Router) {
		r.Get("/", Compresser(srv.handlerGetAll))
	})
	mainRouter.Route("/value", func(r chi.Router) {
		r.Post("/", Compresser(srv.handlerGetMetricJSON))
		r.Get("/{type}/{name}", Compresser(srv.handlerGetMetric))
	})
	mainRouter.Route("/update", func(r chi.Router) {
		r.Post("/", Compresser(srv.handlerUpdateJSON))
		r.Post("/{type}/{name}/{val}", Compresser(srv.handlerUpdateDirect))
	})
	log.Println(http.ListenAndServe(srv.Cfg.SrvAddr, mainRouter))
}
