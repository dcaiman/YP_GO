package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"reflect"
	"runtime"
	"strconv"
	"syscall"
	"time"
)

type Gauge struct {
	Name, Type string
	Value      float64
}

type Counter struct {
	Name, Type string
	Value      int64
}

func NewGauge(name string) Gauge {
	g := Gauge{
		Name: name,
		Type: "gauge",
	}
	return g
}

func NewCounter(name string) Counter {
	c := Counter{
		Name: name,
		Type: "counter",
	}
	return c
}

func (g *Gauge) SetRuntimeValue() bool {
	m := &runtime.MemStats{}
	runtime.ReadMemStats(m)
	tmp := reflect.Indirect(reflect.ValueOf(m)).FieldByName(g.Name).Convert(reflect.TypeOf(g.Value)).Float()
	if g.Value != tmp {
		g.Value = tmp
		return true
	}
	return false
}

func (g *Gauge) SetRandomValue(rate float64) {
	g.Value = rate * rand.Float64()
}

func (g *Gauge) SendMetric(srv, cont string) {
	postReq(srv+"/update/"+g.Type+"/"+g.Name+"/"+strconv.FormatFloat(g.Value, 'f', 3, 64), cont)
}

func (c *Counter) SendMetric(srv, cont string) {
	postReq(srv+"/update/"+c.Type+"/"+c.Name+"/"+strconv.FormatInt(c.Value, 10), cont)
}

func postReq(addr, cont string) error {
	res, err := http.Post(addr, cont, nil)
	if err != nil {
		fmt.Printf("err: %v\n", err)
	}
	defer res.Body.Close()
	fmt.Println(res.Status, res.Request.URL)
	return nil
}

var runtimeMetrics = [...]Gauge{
	NewGauge("Alloc"),
	NewGauge("BuckHashSys"),
	NewGauge("Frees"),
	NewGauge("GCCPUFraction"),
	NewGauge("GCSys"),
	NewGauge("HeapAlloc"),
	NewGauge("HeapIdle"),
	NewGauge("HeapInuse"),
	NewGauge("HeapObjects"),
	NewGauge("HeapReleased"),
	NewGauge("HeapSys"),
	NewGauge("LastGC"),
	NewGauge("Lookups"),
	NewGauge("MCacheInuse"),
	NewGauge("MCacheSys"),
	NewGauge("MSpanInuse"),
	NewGauge("MSpanSys"),
	NewGauge("Mallocs"),
	NewGauge("NextGC"),
	NewGauge("NumForcedGC"),
	NewGauge("NumGC"),
	NewGauge("OtherSys"),
	NewGauge("PauseTotalNs"),
	NewGauge("StackInuse"),
	NewGauge("StackSys"),
	NewGauge("Sys"),
	NewGauge("TotalAlloc"),
}

var PollCount Counter = NewCounter("PollCounter")
var RandomValue Gauge = NewGauge("RandomValue")

var pollInterval = 2 * time.Second
var reportInterval = 10 * time.Second

var contentType = "text/plain"

var srvAddr = "http://127.0.0.1:8080"

func main() {
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
			fmt.Println("EXIT")
			os.Exit(1)
		}
	}
}

func poll() {
	PollCount.Value = 0
	for i := range runtimeMetrics {
		poll := runtimeMetrics[i].SetRuntimeValue()
		if poll {
			PollCount.Value++
		}
	}
	RandomValue.SetRandomValue(100)
	fmt.Println(PollCount)
}

func report() {
	go PollCount.SendMetric(srvAddr, contentType)
	go RandomValue.SendMetric(srvAddr, contentType)
	for i := range runtimeMetrics {
		go runtimeMetrics[i].SendMetric(srvAddr, contentType)
	}
}
