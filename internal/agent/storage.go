package agent

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"reflect"
	"runtime"
	"strconv"
	"strings"

	"github.com/dcaiman/YP_GO/internal/clog"
	"github.com/dcaiman/YP_GO/internal/metric"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
)

const (
	RandomValue    = "RandomValue"
	TotalMemory    = "TotalMemory"
	FreeMemory     = "FreeMemory"
	CPUutilization = "CPUutilization"
)

func (agn *AgentConfig) prepareStorage() error {
	if err := agn.createMetrics(agn.RuntimeGauges[:], Gauge); err != nil {
		return clog.ToLog(clog.FuncName(), err)
	}
	if err := agn.createMetrics(agn.CustomGauges[:], Gauge); err != nil {
		return clog.ToLog(clog.FuncName(), err)
	}
	if err := agn.createMetrics(agn.Counters[:], Counter); err != nil {
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
	for i := range agn.RuntimeGauges {
		val := getRuntimeMetricValue(agn.RuntimeGauges[i])
		m := metric.Metric{
			ID:    agn.RuntimeGauges[i],
			MType: Gauge,
			Value: &val,
		}
		if err := agn.Storage.UpdateMetric(m); err != nil {
			return clog.ToLog(clog.FuncName(), err)
		}
	}
	for i := range agn.CustomGauges {
		val, err := getCustomMetricValue(agn.CustomGauges[i])
		if err != nil {
			return clog.ToLog(clog.FuncName(), err)
		}
		m := metric.Metric{
			ID:    agn.CustomGauges[i],
			MType: Gauge,
			Value: &val,
		}
		if err := agn.Storage.UpdateMetric(m); err != nil {
			return clog.ToLog(clog.FuncName(), err)
		}
	}
	for i := range agn.Counters {
		var del int64 = 1
		m := metric.Metric{
			ID:    agn.Counters[i],
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
	for i := range agn.RuntimeGauges {
		go func(i int) {
			if err := agn.sendMetric(agn.RuntimeGauges[i]); err != nil {
				log.Println(clog.ToLog(clog.FuncName(), err))
			}
		}(i)
	}
	for i := range agn.CustomGauges {
		go func(i int) {
			if err := agn.sendMetric(agn.CustomGauges[i]); err != nil {
				log.Println(clog.ToLog(clog.FuncName(), err))
			}
		}(i)
	}
	for i := range agn.Counters {
		go func(i int) {
			if err := agn.sendMetric(agn.Counters[i]); err != nil {
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

func getCustomMetricValue(name string) (float64, error) {
	switch name {
	case RandomValue:
		return 100 * rand.Float64(), nil
	case TotalMemory:
		vm, err := mem.VirtualMemory()
		if err != nil {
			return 0, clog.ToLog(clog.FuncName(), err)
		}
		return float64(vm.Total), nil
	case FreeMemory:
		vm, err := mem.VirtualMemory()
		if err != nil {
			return 0, clog.ToLog(clog.FuncName(), err)
		}
		return float64(vm.Free), nil
	}
	if strings.Contains(name, CPUutilization) {
		vals, err := cpu.Percent(0, true)
		if err != nil {
			return 0, clog.ToLog(clog.FuncName(), err)
		}
		num, err := strconv.ParseInt(strings.TrimPrefix(name, CPUutilization), 10, 64)
		if err != nil {
			return 0, clog.ToLog(clog.FuncName(), err)
		}
		if int(num) > len(vals) {
			return 0, clog.ToLog(clog.FuncName(), errors.New("cannot get <"+name+">: core number error"))
		}
		fmt.Println(name, num, vals, vals[num-1])
		return vals[num-1], nil
	}
	return 0, clog.ToLog(clog.FuncName(), errors.New("cannot get: unsupported metric <"+name+">"))
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
	if err := agn.createMetrics(agn.Counters[:], Counter); err != nil {
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
