package server

import (
	"flag"
	"log"
	"net/http"
	"time"

	_ "github.com/jackc/pgx/v4/stdlib"

	"github.com/dcaiman/YP_GO/internal/clog"
	"github.com/dcaiman/YP_GO/internal/internalstorage"
	"github.com/dcaiman/YP_GO/internal/metric"
	"github.com/dcaiman/YP_GO/internal/pgxstorage"

	"github.com/caarlos0/env"
	"github.com/go-chi/chi/v5"
)

type EnvConfig struct {
	SrvAddr       string        `env:"ADDRESS"`
	StoreFile     string        `env:"STORE_FILE"`
	DBAddr        string        `env:"DATABASE_DSN"`
	StoreInterval time.Duration `env:"STORE_INTERVAL"`
	InitDownload  bool          `env:"RESTORE"`
	HashKey       string        `env:"KEY"`

	SyncUpload chan struct{}

	EnvConfig bool
	ArgConfig bool
	DropDB    bool
}

type ServerConfig struct {
	Storage metric.MStorage
	Cfg     EnvConfig
}

func RunServer(srv *ServerConfig) {
	if srv.Cfg.DBAddr != "" {
		dbStorage, err := pgxstorage.New(srv.Cfg.DBAddr, srv.Cfg.DropDB)
		if err != nil {
			log.Println(clog.ToLog(clog.FuncName(), err))
		}
		defer dbStorage.Close()
		srv.Storage = dbStorage
	} else if srv.Cfg.StoreFile != "" {
		fileStorage := internalstorage.New(srv.Cfg.StoreFile, srv.Cfg.HashKey)

		if srv.Cfg.InitDownload {
			err := fileStorage.DownloadStorage()
			if err != nil {
				log.Println(clog.ToLog(clog.FuncName(), err))
			}
		}
		if srv.Cfg.StoreInterval != 0 {
			go func() {
				uploadTimer := time.NewTicker(srv.Cfg.StoreInterval)
				for {
					<-uploadTimer.C
					if err := fileStorage.UploadStorage(); err != nil {
						log.Println(clog.ToLog(clog.FuncName(), err))
					}
				}
			}()
		} else {
			srv.Cfg.SyncUpload = make(chan struct{})
			go func(c chan struct{}) {
				for {
					<-c
					if err := fileStorage.UploadStorage(); err != nil {
						log.Println(clog.ToLog(clog.FuncName(), err))
					}
				}
			}(srv.Cfg.SyncUpload)
		}
		srv.Storage = fileStorage
	}

	log.Println("SERVER CONFIG: ", srv.Cfg)

	mainRouter := chi.NewRouter()
	mainRouter.Use(Compresser)
	mainRouter.Route("/", func(r chi.Router) {
		r.Get("/", srv.handlerGetAll)
	})
	mainRouter.Route("/value", func(r chi.Router) {
		r.Post("/", srv.handlerGetMetricJSON)
		r.Get("/{type}/{name}", srv.handlerGetMetric)
	})
	mainRouter.Route("/update", func(r chi.Router) {
		r.Post("/", srv.handlerUpdateJSON)
		r.Post("/{type}/{name}/{val}", srv.handlerUpdateDirect)
	})
	mainRouter.Route("/updates", func(r chi.Router) {
		r.Post("/", srv.handlerUpdateBatch)
	})
	mainRouter.Route("/ping", func(r chi.Router) {
		r.Get("/", srv.handlerCheckDBConnection)
	})
	log.Println(http.ListenAndServe(srv.Cfg.SrvAddr, mainRouter))
}

func (srv *ServerConfig) GetExternalConfig() error {
	if srv.Cfg.ArgConfig {
		flag.BoolVar(&srv.Cfg.InitDownload, "r", srv.Cfg.InitDownload, "initial download flag")
		flag.StringVar(&srv.Cfg.StoreFile, "f", srv.Cfg.StoreFile, "storage file destination")
		flag.StringVar(&srv.Cfg.SrvAddr, "a", srv.Cfg.SrvAddr, "server address")
		flag.DurationVar(&srv.Cfg.StoreInterval, "i", srv.Cfg.StoreInterval, "store interval")
		flag.StringVar(&srv.Cfg.HashKey, "k", srv.Cfg.HashKey, "hash key")
		flag.StringVar(&srv.Cfg.DBAddr, "d", srv.Cfg.DBAddr, "database address")
		flag.Parse()
	}
	if srv.Cfg.EnvConfig {
		if err := env.Parse(&srv.Cfg); err != nil {
			return clog.ToLog(clog.FuncName(), err)
		}
	}
	return nil
}
