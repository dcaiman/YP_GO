//
package server

import (
	"database/sql"
	"errors"
	"flag"
	"log"
	"net/http"
	"time"

	_ "github.com/jackc/pgx/v4/stdlib"

	"github.com/dcaiman/YP_GO/internal/metrics"

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

	SyncUpload bool

	EnvConfig bool
	ArgConfig bool

	DB *sql.DB
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
		flag.StringVar(&srv.Cfg.DBAddr, "d", srv.Cfg.DBAddr, "database address")
		flag.Parse()
	}
	if srv.Cfg.EnvConfig {
		if err := env.Parse(&srv.Cfg); err != nil {
			log.Println(err.Error())
		}
	}

	if false {
		//if srv.Cfg.DBAddr != "" {
		if DB, err := initDBStorage(srv); err != nil {
			log.Println(err)
		} else {
			srv.Cfg.DB = DB
			defer srv.Cfg.DB.Close()
		}
	} else if srv.Cfg.StoreFile != "" {

		if err := initFileStorage(srv); err != nil {
			log.Println(err)
		}
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
	mainRouter.Route("/ping", func(r chi.Router) {
		r.Get("/", Compresser(srv.handlerCheckDBConnection))
	})
	log.Println(http.ListenAndServe(srv.Cfg.SrvAddr, mainRouter))
}

func initDBStorage(srv *ServerConfig) (*sql.DB, error) {
	var DB *sql.DB
	var err error
	if srv.Cfg.InitDownload {
		DB, err = sql.Open("pgx", srv.Cfg.DBAddr)
		if err != nil {
			return nil, err
		}
	}
	if srv.Cfg.StoreInterval != 0 {
		go func() {
			uploadTimer := time.NewTicker(srv.Cfg.StoreInterval)
			for {
				<-uploadTimer.C
				err = uploadStorage()
				if err != nil {
					log.Println(err)
				}
			}
		}()
	} else {
		srv.Cfg.SyncUpload = true
	}
	return DB, nil
}

func initFileStorage(srv *ServerConfig) error {
	if srv.Cfg.InitDownload {
		err := srv.Storage.DownloadStorage(srv.Cfg.StoreFile)
		if err != nil {
			return err
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
	return nil
}

func uploadStorage() error {
	return errors.New("UPLOAD TO DB IS NOT IMPLEMENTED")
}
