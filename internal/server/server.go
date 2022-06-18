//
package server

import (
	"log"
	"net/http"

	"github.com/caarlos0/env"
	"github.com/go-chi/chi/v5"
)

type EnvConfig struct {
	SrvAddr string `env:"ADDRESS"`
}

var cfg EnvConfig
var storage Metrics

func RunServer() {
	storage = Metrics{
		Gauges:   map[string]float64{},
		Counters: map[string]int64{},
	}
	cfg = EnvConfig{
		SrvAddr: "127.0.0.1:8080",
	}
	if err := env.Parse(&cfg); err != nil {
		log.Println(err.Error())
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
