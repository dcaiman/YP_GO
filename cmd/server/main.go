package main

import (
	"log"
	"time"

	"github.com/dcaiman/YP_GO/internal/clog"
	"github.com/dcaiman/YP_GO/internal/server"
)

func main() {
	srv := server.ServerConfig{
		Cfg: server.EnvConfig{
			SrvAddr: "127.0.0.1:8080",
			//DBAddr:        "postgresql://postgres:1@127.0.0.1:5432",
			StoreInterval: 0 * time.Second,
			StoreFile:     "./tmp/metricStorage.json",
			HashKey:       "key",
			InitDownload:  true,

			ArgConfig: true,
			EnvConfig: true,
			DropDB:    false,
		},
	}
	if err := srv.GetExternalConfig(); err != nil {
		log.Println(clog.ToLog(clog.FuncName(), err))
		return
	}
	server.RunServer(&srv)
}
