package agent

import (
	"log"
	"math/rand"
	"reflect"
	"runtime"

	"github.com/dcaiman/YP_GO/internal/metric"
)

func (agn *AgentConfig) prepareStorage() {
	for i := range runtimeGauges {
		m := metric.Metric{
			ID:    runtimeGauges[i],
			MType: Gauge,
		}
		m.UpdateHash(agn.Cfg.HashKey)
		agn.Storage.NewMetric(m)
	}
	for i := range customGauges {
		m := metric.Metric{
			ID:    customGauges[i],
			MType: Gauge,
		}
		m.UpdateHash(agn.Cfg.HashKey)
		agn.Storage.NewMetric(m)
	}
	for i := range counters {
		m := metric.Metric{
			ID:    counters[i],
			MType: Counter,
		}
		m.UpdateHash(agn.Cfg.HashKey)
		agn.Storage.NewMetric(m)
	}
}

func (agn *AgentConfig) poll() {
	for i := range runtimeGauges {
		if err := agn.Storage.UpdateValue(runtimeGauges[i], getRuntimeMetric(runtimeGauges[i])); err != nil {
			log.Println(err.Error())
		}
	}
	for i := range customGauges {
		if err := agn.Storage.UpdateValue(customGauges[i], 100*rand.Float64()); err != nil {
			log.Println(err.Error())
		}
	}
	for i := range counters {
		if err := agn.Storage.IncreaseDelta(counters[i]); err != nil {
			log.Println(err.Error())
		}
	}
}

func (agn *AgentConfig) report(sendBatch bool) {
	if sendBatch {
		if err := agn.sendBatch(); err != nil {
			log.Println(err.Error())
			return
		}
		return
	}
	for i := range runtimeGauges {
		go agn.sendMetric(runtimeGauges[i])
	}
	for i := range customGauges {
		go agn.sendMetric(customGauges[i])
	}
	for i := range counters {
		go agn.sendMetric(counters[i])
	}

}

func getRuntimeMetric(name string) float64 {
	mem := &runtime.MemStats{}
	runtime.ReadMemStats(mem)
	return reflect.Indirect(reflect.ValueOf(mem)).FieldByName(name).Convert(reflect.TypeOf(0.0)).Float()
}

func (agn *AgentConfig) getStorageBatch(resetCounters bool) ([]byte, error) {
	allMetrics, err := agn.Storage.GetAllMetrics()
	if err != nil {
		return nil, err
	}

	var mj []byte
	for i := range allMetrics {
		tmp, err := allMetrics[i].GetJSON()
		if err != nil {
			return nil, err
		}
		mj = append(mj, tmp...)
		mj = append(mj, []byte(",")...)
		if resetCounters && allMetrics[i].MType == Counter {
			if err := agn.Storage.ResetDelta(allMetrics[i].ID); err != nil {
				return nil, err
			}
		}
	}
	return mj, nil
}
