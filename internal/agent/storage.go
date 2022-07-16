package agent

import (
	"encoding/json"
	"errors"
	"log"
	"math/rand"
	"reflect"
	"runtime"

	"github.com/dcaiman/YP_GO/internal/clog"
	"github.com/dcaiman/YP_GO/internal/metric"
)

func (agn *AgentConfig) prepareStorage() error {
	if err := agn.createMetrics(runtimeGauges[:], Gauge); err != nil {
		return clog.ToLog(clog.FuncName(), err)
	}
	if err := agn.createMetrics(customGauges[:], Gauge); err != nil {
		return clog.ToLog(clog.FuncName(), err)
	}
	if err := agn.createMetrics(counters[:], Counter); err != nil {
		return clog.ToLog(clog.FuncName(), err)
	}
	return nil
}

func (agn *AgentConfig) createMetrics(mNames []string, mType string) error {
	for i := range mNames {
		m := metric.Metric{
			ID:    mNames[i],
			MType: mType,
		}
		if err := agn.Storage.UpdateMetric(m); err != nil {
			return clog.ToLog(clog.FuncName(), err)
		}
	}
	return nil
}

func (agn *AgentConfig) poll() error {
	for i := range runtimeGauges {
		val := getRuntimeMetricValue(runtimeGauges[i])
		m := metric.Metric{
			ID:    runtimeGauges[i],
			MType: Gauge,
			Value: &val,
		}
		if err := agn.Storage.UpdateMetric(m); err != nil {
			return clog.ToLog(clog.FuncName(), err)
		}
	}
	for i := range customGauges {
		val := 100 * rand.Float64()
		m := metric.Metric{
			ID:    customGauges[i],
			MType: Gauge,
			Value: &val,
		}
		if err := agn.Storage.UpdateMetric(m); err != nil {
			return clog.ToLog(clog.FuncName(), err)
		}
	}
	for i := range counters {
		var del int64 = 1
		m := metric.Metric{
			ID:    counters[i],
			MType: Counter,
			Delta: &del,
		}
		if err := agn.Storage.UpdateMetric(m); err != nil {
			return clog.ToLog(clog.FuncName(), err)
		}
	}
	return nil
}

func (agn *AgentConfig) report(sendBatch bool) error {
	if sendBatch {
		if err := agn.sendBatch(); err != nil {
			return clog.ToLog(clog.FuncName(), err)
		}
		return nil
	}
	for i := range runtimeGauges {
		go func(i int) {
			if err := agn.sendMetric(runtimeGauges[i]); err != nil {
				log.Println(clog.ToLog(clog.FuncName(), err))
			}
		}(i)
	}
	for i := range customGauges {
		go func(i int) {
			if err := agn.sendMetric(customGauges[i]); err != nil {
				log.Println(clog.ToLog(clog.FuncName(), err))
			}
		}(i)
	}
	for i := range counters {
		go func(i int) {
			if err := agn.sendMetric(counters[i]); err != nil {
				log.Println(clog.ToLog(clog.FuncName(), err))
			}
		}(i)
	}
	return nil
}

func getRuntimeMetricValue(name string) float64 {
	mem := &runtime.MemStats{}
	runtime.ReadMemStats(mem)
	return reflect.Indirect(reflect.ValueOf(mem)).FieldByName(name).Convert(reflect.TypeOf(0.0)).Float()
}

func (agn *AgentConfig) getStorageBatch() ([]byte, error) {
	allMetrics, err := agn.Storage.GetBatch()
	if err != nil {
		return nil, clog.ToLog(clog.FuncName(), err)
	}

	var mj []byte
	for i := range allMetrics {
		if err := allMetrics[i].UpdateHash(agn.Cfg.HashKey); err != nil {
			return nil, clog.ToLog(clog.FuncName(), err)
		}
	}
	mj, err = json.Marshal(allMetrics)
	if err != nil {
		return nil, clog.ToLog(clog.FuncName(), err)
	}
	if mj == nil {
		return nil, clog.ToLog(clog.FuncName(), errors.New("empty batch"))
	}
	return mj, nil
}

func (agn *AgentConfig) resetCounters() error {
	if err := agn.createMetrics(counters[:], Counter); err != nil {
		return clog.ToLog(clog.FuncName(), err)
	}
	return nil
}

func (agn *AgentConfig) resetCounter(name string) error {
	if err := agn.Storage.UpdateMetric(metric.Metric{
		ID:    name,
		MType: Counter,
	}); err != nil {
		return clog.ToLog(clog.FuncName(), err)
	}
	return nil
}
