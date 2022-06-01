package agent

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var pollInterval = 2 * time.Second
var reportInterval = 10 * time.Second
var contentType = "text/plain"
var srvAddr = "http://127.0.0.1:8080"

func RunAgent() {
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
	resetCounter(counters[0])
	for i := range runtimeGauges {
		polled := updGaugeByRuntimeValue(runtimeGauges[i])
		if polled {
			updCounter(counters[0])
		}
	}
	updGaugeByRandomValue(customGauges[0])
	fmt.Println(getCounter(counters[0]))
}

func report() {
	go sendCounter(srvAddr, contentType, counters[0])
	go sendGauge(srvAddr, contentType, customGauges[0])
	for i := range runtimeGauges {
		go sendGauge(srvAddr, contentType, runtimeGauges[i])
	}
}
