package main

import (
	"time"

	"github.com/dcaiman/YP_GO/internal/server"
)

func main() {
	srv := server.ServerConfig{
		Cfg: server.EnvConfig{
			SrvAddr:       "127.0.0.1:8080",
			DBAddr:        "", //"postgresql://postgres:1@127.0.0.1:5432",
			StoreInterval: 4 * time.Second,
			StoreFile:     "./tmp/metricStorage.json",
			HashKey:       "key",
			InitDownload:  true,

			ArgConfig: true,
			EnvConfig: true,
		},
	}
	server.RunServer(&srv)
}
