package main

import (
	"time"

	"github.com/dcaiman/YP_GO/internal/agent"
	"github.com/dcaiman/YP_GO/internal/metrics"
)

func main() {
	agn := agent.AgentConfig{
		Storage: metrics.MetricStorage{},
		Cfg: agent.EnvConfig{
			PollInterval:   2 * time.Second,
			ReportInterval: 6 * time.Second,
			SrvAddr:        "127.0.0.1:8080",
			HashKey:        "key",
			ArgConfig:      true,
			EnvConfig:      true,
		},
	}
	agent.RunAgent(&agn)
}
