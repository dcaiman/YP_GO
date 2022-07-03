package main

import (
	"github.com/dcaiman/YP_GO/internal/metrics"
	"github.com/dcaiman/YP_GO/internal/server"
)

func main() {
	srv := server.ServerConfig{
		Storage: metrics.MetricStorage{},
		Cfg: server.EnvConfig{
			SrvAddr:       "127.0.0.1:8080",
			DBAddr:        "postgresql://postgres:1@127.0.0.1:5432",
			StoreInterval: 0,
			StoreFile:     "./tmp/metricStorage.json",
			HashKey:       "key",
			InitDownload:  true,

			ArgConfig: true,
			EnvConfig: true,
		},
	}
	server.RunServer(&srv)
}
