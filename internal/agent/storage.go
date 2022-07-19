package agent

import (
	"encoding/json"
	"errors"
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

func (agn *AgentEnv) getRuntimeMetricValue(name string) float64 {
	mem := &runtime.MemStats{}
	runtime.ReadMemStats(mem)
	return reflect.Indirect(reflect.ValueOf(mem)).FieldByName(name).Convert(reflect.TypeOf(0.0)).Float()
}

func (agn *AgentEnv) getCustomMetricValue(name string) (float64, error) {
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
		procUsage, err := cpu.Percent(0, true)
		if err != nil {
			return 0, clog.ToLog(clog.FuncName(), err)
		}
		num, err := strconv.ParseInt(strings.TrimPrefix(name, CPUutilization), 10, 64)
		if err != nil {
			return 0, clog.ToLog(clog.FuncName(), err)
		}
		if int(num) > len(procUsage) {
			return 0, clog.ToLog(clog.FuncName(), errors.New("cannot get <"+name+">: core number error"))
		}
		return procUsage[num-1], nil
	}
	return 0, clog.ToLog(clog.FuncName(), errors.New("cannot get: unsupported metric <"+name+">"))
}

func (agn *AgentEnv) getStorageBatch() ([]byte, error) {
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

func (agn *AgentEnv) resetCounters() error {
	for _, name := range agn.Counters {
		m := metric.Metric{
			ID:    name,
			MType: Counter,
		}
		if err := agn.Storage.UpdateMetric(m); err != nil {
			return clog.ToLog(clog.FuncName(), err)
		}
	}
	return nil
}

func (agn *AgentEnv) resetCounter(name string) error {
	if err := agn.Storage.UpdateMetric(metric.Metric{
		ID:    name,
		MType: Counter,
	}); err != nil {
		return clog.ToLog(clog.FuncName(), err)
	}
	return nil
}
