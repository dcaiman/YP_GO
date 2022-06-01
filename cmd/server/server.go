package server

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

var srvAddr = "127.0.0.1:8080"

func RunServer() {
	mainRouter := chi.NewRouter()
	mainRouter.Route("/", func(r chi.Router) {
		r.Get("/", hdlrGetAll)
	})
	mainRouter.Route("/value", func(r chi.Router) {
		r.Get("/{type}", hdlrGetMetricsByType)
		r.Get("/{type}/{name}", hdlrGetMetric)
	})
	mainRouter.Route("/update", func(r chi.Router) {
		r.Post("/{type}/{name}/{val}", hdlrUpdate)
	})

	http.ListenAndServe(srvAddr, mainRouter)
}
