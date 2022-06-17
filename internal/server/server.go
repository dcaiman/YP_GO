//
package server

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

var srvAddr = "127.0.0.1:8080"
var storage Metrics

func RunServer() {
	storage = Metrics{
		Gauges:   map[string]float64{},
		Counters: map[string]int64{},
	}
	mainRouter := chi.NewRouter()
	mainRouter.Route("/", func(r chi.Router) {
		r.Get("/", handlerGetAll)
	})
	mainRouter.Route("/value", func(r chi.Router) {
		r.Get("/{type}", handlerGetMetricsByType)
		r.Get("/{type}/{name}", handlerGetMetric)
	})
	mainRouter.Route("/update", func(r chi.Router) {
		r.Post("/{type}/{name}/{val}", handlerUpdate)
	})

	http.ListenAndServe(srvAddr, mainRouter)
}
