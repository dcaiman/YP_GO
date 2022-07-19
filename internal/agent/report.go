package agent

import (
	"log"
	"time"

	"github.com/dcaiman/YP_GO/internal/clog"
)

func (agn *AgentEnv) report() {
	reportTimer := time.NewTicker(agn.Cfg.ReportInterval)
	for {
		<-reportTimer.C
		if agn.Cfg.SendBatch {
			if err := agn.sendBatch(); err != nil {
				log.Println(clog.ToLog(clog.FuncName(), err))
			}
		} else {
			agn.reportMetrics(agn.RuntimeGauges)
			agn.reportMetrics(agn.CustomGauges)
			agn.reportMetrics(agn.Counters)
		}
	}
}

func (agn *AgentEnv) reportMetrics(names []string) {
	for i := range names {
		go func(i int) {
			if err := agn.sendMetric(names[i]); err != nil {
				log.Println(clog.ToLog(clog.FuncName(), err))
			}
		}(i)
	}
}
