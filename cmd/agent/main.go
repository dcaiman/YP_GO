package main

import (
	"fmt"
	"log"
	"runtime"
	"time"

	"github.com/dcaiman/YP_GO/internal/agent"
	"github.com/dcaiman/YP_GO/internal/clog"
)

func main() {
	runtimeGauges := []string{
		"Alloc",
		"BuckHashSys",
		"Frees",
		"GCCPUFraction",
		"GCSys",
		"HeapAlloc",
		"HeapIdle",
		"HeapInuse",
		"HeapObjects",
		"HeapReleased",
		"HeapSys",
		"LastGC",
		"Lookups",
		"MCacheInuse",
		"MCacheSys",
		"MSpanInuse",
		"MSpanSys",
		"Mallocs",
		"NextGC",
		"NumForcedGC",
		"NumGC",
		"OtherSys",
		"PauseTotalNs",
		"StackInuse",
		"StackSys",
		"Sys",
		"TotalAlloc",
	}
	customGauges := []string{
		"RandomValue",
		"TotalMemory",
		"FreeMemory",
	}
	counters := []string{
		"PollCount",
	}
	for i := 1; i <= runtime.NumCPU(); i++ {
		customGauges = append(customGauges, "CPUutilization"+fmt.Sprint(i))
	}

	agn := agent.AgentEnv{
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
		RuntimeGauges: runtimeGauges,
		CustomGauges:  customGauges,
		Counters:      counters,
	}
	if err := agn.GetExternalConfig(); err != nil {
		log.Println(clog.ToLog(clog.FuncName(), err))
		return
	}
	agent.RunAgent(&agn)
}
