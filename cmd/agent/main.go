package main

import (
	"log"
	"time"

	"github.com/dcaiman/YP_GO/internal/agent"
	"github.com/dcaiman/YP_GO/internal/clog"
)

func main() {
	agn := agent.AgentConfig{
		Cfg: agent.EnvConfig{
			CType:          agent.JSONCT,
			PollInterval:   2 * time.Second,
			ReportInterval: 6 * time.Second,
			SrvAddr:        "127.0.0.1:8080",
			HashKey:        "key1",
			ArgConfig:      true,
			EnvConfig:      true,
			SendBatch:      true,
		},
	}
	if err := agn.GetExternalConfig(); err != nil {
		log.Println(clog.ToLog(clog.FuncName(), err))
		return
	}
	agent.RunAgent(&agn)
}
