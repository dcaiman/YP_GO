package main

import (
	"time"

	"github.com/dcaiman/YP_GO/internal/agent"
)

func main() {
	agn := agent.AgentConfig{
		Cfg: agent.EnvConfig{
			CType:          agent.JSONCT,
			PollInterval:   2 * time.Second,
			ReportInterval: 6 * time.Second,
			SrvAddr:        "127.0.0.1:8080",
			HashKey:        "key",
			ArgConfig:      true,
			EnvConfig:      true,
			SendBatch:      true,
		},
	}
	agent.RunAgent(&agn)
}
