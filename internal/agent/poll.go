package agent

import (
	"log"
	"time"

	"github.com/dcaiman/YP_GO/internal/clog"
	"github.com/dcaiman/YP_GO/internal/metric"
)

func (agn *AgentEnv) poll() {
	pollTimer := time.NewTicker(agn.Cfg.PollInterval)
	for {
		<-pollTimer.C
		agn.pollRuntimeGauges()
		agn.pollCustomGauges()
		agn.pollCounters()
	}
}

func (agn *AgentEnv) pollRuntimeGauges() {
	for i := range agn.RuntimeGauges {
		go func(name string) {
			val := agn.getRuntimeMetricValue(name)
			m := metric.Metric{
				ID:    name,
				MType: Gauge,
				Value: &val,
			}
			if err := agn.Storage.UpdateMetric(m); err != nil {
				log.Println(clog.ToLog(clog.FuncName(), err))
				return
			}
		}(agn.RuntimeGauges[i])
	}
}

func (agn *AgentEnv) pollCustomGauges() {
	for i := range agn.CustomGauges {
		go func(name string) {
			val, err := agn.getCustomMetricValue(name)
			if err != nil {
				log.Println(clog.ToLog(clog.FuncName(), err))
				return
			}
			m := metric.Metric{
				ID:    name,
				MType: Gauge,
				Value: &val,
			}
			if err := agn.Storage.UpdateMetric(m); err != nil {
				log.Println(clog.ToLog(clog.FuncName(), err))
				return
			}
		}(agn.CustomGauges[i])
	}
}

func (agn *AgentEnv) pollCounters() {
	for i := range agn.Counters {
		go func(name string) {
			var del int64 = 1
			m := metric.Metric{
				ID:    name,
				MType: Counter,
				Delta: &del,
			}
			if err := agn.Storage.UpdateMetric(m); err != nil {
				log.Println(clog.ToLog(clog.FuncName(), err))
				return
			}
		}(agn.Counters[i])
	}
}
