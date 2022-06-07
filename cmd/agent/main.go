package agent

import (
	"errors"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"reflect"
	"runtime"
	"strconv"
	"sync"
	"syscall"
	"time"
)

var pollInterval = 2 * time.Second
var reportInterval = 10 * time.Second
var contentType = "text/plain"
var srvAddr = "http://127.0.0.1:8080"
var storage Metrics

func RunAgent() {
	storage = Metrics{
		Gauges:   map[string]float64{},
		Counters: map[string]int64{},
	}

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	pollTimer := time.NewTicker(pollInterval)
	reportTimer := time.NewTicker(reportInterval)
	for {
		select {
		case <-pollTimer.C:
			poll()
		case <-reportTimer.C:
			report()
		case <-signalCh:
			log.Println("EXIT")
			os.Exit(0)
		}
	}
}

var runtimeGauges = [...]string{
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

var customGauges = [...]string{
	"RandomValue",
}

var counters = [...]string{
	"PollCount",
}

type Metrics struct {
	sync.RWMutex
	Gauges   map[string]float64
	Counters map[string]int64
}

func (m *Metrics) getGauge(name string) (float64, error) {
	m.RLock()
	defer m.RUnlock()
	if val, ok := m.Gauges[name]; ok {
		return val, nil
	}
	err := errors.New("cannot get: no such gauge <" + name + ">")
	log.Println(err.Error())
	return 0, err
}

func (m *Metrics) getCounter(name string) (int64, error) {
	m.RLock()
	defer m.RUnlock()
	if val, ok := m.Counters[name]; ok {
		return val, nil
	}
	err := errors.New("cannot get: no such counter <" + name + ">")
	log.Println(err.Error())
	return 0, err
}

func (m *Metrics) updateGaugeByRuntimeValue(name string) {
	m.Lock()
	defer m.Unlock()
	mem := &runtime.MemStats{}
	runtime.ReadMemStats(mem)
	m.Gauges[name] = reflect.Indirect(reflect.ValueOf(mem)).FieldByName(name).Convert(reflect.TypeOf(m.Gauges[name])).Float()
}

func (m *Metrics) updateGaugeByRandomValue(name string) {
	m.Lock()
	defer m.Unlock()
	m.Gauges[name] = 100 * rand.Float64()
}

func (m *Metrics) updateCounter(name string) {
	m.Lock()
	defer m.Unlock()
	m.Counters[name] += 1
}

func (m *Metrics) sendGauge(srvAddr, contentType, metricName string) error {
	val, err := m.getGauge(metricName)
	if err != nil {
		log.Println(err.Error())
		return err
	}
	if err = sendPostReqest(srvAddr+"/update/gauge/"+metricName+"/"+strconv.FormatFloat(val, 'f', 3, 64), contentType); err != nil {
		log.Println(err.Error())
		return err
	}
	return nil
}

func (m *Metrics) sendCounter(srvAddr, contentType, metricName string) error {
	val, err := m.getCounter(metricName)
	if err != nil {
		log.Println(err.Error())
		return err
	}
	if err = sendPostReqest(srvAddr+"/update/counter/"+metricName+"/"+strconv.FormatInt(val, 10), contentType); err != nil {
		log.Println(err.Error())
		return err
	}
	return nil
}

func sendPostReqest(addr, cont string) error {
	res, err := http.Post(addr, cont, nil)
	if err != nil {
		log.Println(err.Error())
		return err
	}
	defer res.Body.Close()
	log.Println(res.Status, res.Request.URL)
	return nil
}

func poll() {
	for i := range runtimeGauges {
		storage.updateGaugeByRuntimeValue(runtimeGauges[i])
	}
	storage.updateGaugeByRandomValue(customGauges[0])
	storage.updateCounter(counters[0])
}

func report() {
	go storage.sendCounter(srvAddr, contentType, counters[0])
	go storage.sendGauge(srvAddr, contentType, customGauges[0])
	for i := range runtimeGauges {
		go storage.sendGauge(srvAddr, contentType, runtimeGauges[i])
	}
}
