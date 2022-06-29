package server

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"reflect"
	"strconv"
	"text/template"

	"github.com/dcaiman/YP_GO/internal/metrics"

	"github.com/go-chi/chi/v5"
)

const (
	templateHandlerGetAll = "GAUGES LIST:\n{{range $v := .Gauges}}\n{{$v}}{{end}}\n\nCOUNTERS LIST:\n{{range $v := .Counters}}\n{{$v}}{{end}}"
	templateGauges        = "GAUGES LIST:\n{{range $v := .}}\n{{$v}}{{end}}"
	templateCounters      = "COUNTERS LIST:\n{{range $v := .}}\n{{$v}}{{end}}"
)

func handlerUpdateJSON(w http.ResponseWriter, r *http.Request) {
	var resHash string
	content, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	mj := metrics.MetricJSON{}
	if err := json.Unmarshal(content, &mj); err != nil {
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	switch mj.MType {
	case metrics.Gauge:
		serverHash, err := storage.MetricJSONHash(fmt.Sprintf("%s:gauge:%f", mj.ID, *mj.Value), storage.EncryptingKey)
		if err != nil {
			log.Println(err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		agentHash, err := hex.DecodeString(mj.Hash)
		if err != nil {
			log.Println(err.Error())
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if !reflect.DeepEqual(serverHash, agentHash) {
			err := errors.New("inconsistent hash")
			log.Println(err.Error())
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		resHash = hex.EncodeToString(serverHash)
		storage.UpdateGaugeByValue(mj.ID, *mj.Value)
	case metrics.Counter:
		serverHash, err := storage.MetricJSONHash(fmt.Sprintf("%s:counter:%d", mj.ID, *mj.Delta), storage.EncryptingKey)
		if err != nil {
			log.Println(err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		agentHash, err := hex.DecodeString(mj.Hash)
		if err != nil {
			log.Println(err.Error())
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if !reflect.DeepEqual(serverHash, agentHash) {
			err := errors.New("inconsistent hash")
			log.Println(err.Error())
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		resHash = hex.EncodeToString(serverHash)
		storage.IncreaseCounter(mj.ID, *mj.Delta)
	default:
		err := errors.New("cannot update: no such metric type <" + mj.ID + ">")
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusNotImplemented)
		return
	}
	if cfg.SyncUpload {
		storage.UploadStorage(cfg.StoreFile)
	}
	w.Header().Set("Hash", resHash)
}

func handlerUpdateDirect(w http.ResponseWriter, r *http.Request) {
	metricType := chi.URLParam(r, "type")
	metricName := chi.URLParam(r, "name")
	metricVal := chi.URLParam(r, "val")
	switch metricType {
	case metrics.Gauge:
		val, err := strconv.ParseFloat(metricVal, 64)
		if err != nil {
			log.Println(err.Error())
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		storage.UpdateGaugeByValue(metricName, val)
	case metrics.Counter:
		val, err := strconv.ParseInt(metricVal, 10, 64)
		if err != nil {
			log.Println(err.Error())
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		storage.IncreaseCounter(metricName, val)
	default:
		err := errors.New("cannot update: no such metric type <" + metricType + ">")
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusNotImplemented)
		return
	}
	if cfg.SyncUpload {
		storage.UploadStorage(cfg.StoreFile)
	}
}

func handlerGetAll(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	t, err := template.New("").Parse(templateHandlerGetAll)
	if err != nil {
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	t.Execute(w, struct {
		Gauges, Counters []string
	}{
		Gauges:   storage.GetGauges(),
		Counters: storage.GetCounters(),
	})
}

func handlerGetMetricJSON(w http.ResponseWriter, r *http.Request) {
	content, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	mj := metrics.MetricJSON{}
	if err := json.Unmarshal(content, &mj); err != nil {
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	switch mj.MType {
	case metrics.Gauge:
		val, err := storage.GetGauge(mj.ID)
		if err != nil {
			log.Println(err.Error())
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		mj.Value = &val
	case metrics.Counter:
		val, err := storage.GetCounter(mj.ID)
		if err != nil {
			log.Println(err.Error())
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		mj.Delta = &val
	default:
		err := errors.New("cannot get: no such metrics type <" + mj.MType + ">")
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusNotImplemented)
		return
	}
	mjRes, err := json.Marshal(mj)
	if err != nil {
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", metrics.JSONCT)
	w.Write(mjRes)
}

func handlerGetMetric(w http.ResponseWriter, r *http.Request) {
	metricType := chi.URLParam(r, "type")
	metricName := chi.URLParam(r, "name")
	switch metricType {
	case metrics.Gauge:
		if metricVal, err := storage.GetGauge(metricName); err == nil {
			w.Write([]byte(strconv.FormatFloat(metricVal, 'f', 3, 64)))
		} else {
			log.Println(err.Error())
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
	case metrics.Counter:
		if metricVal, err := storage.GetCounter(metricName); err == nil {
			w.Write([]byte(strconv.FormatInt(metricVal, 10)))
		} else {
			log.Println(err.Error())
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
	default:
		err := errors.New("cannot get: no such metrics type <" + metricType + ">")
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusNotImplemented)
		return
	}
}

func handlerGetMetricsByType(w http.ResponseWriter, r *http.Request) {
	metricType := chi.URLParam(r, "type")
	switch metricType {
	case metrics.Gauge:
		t, err := template.New("").Parse(templateGauges)
		if err != nil {
			log.Println(err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		t.Execute(w, storage.GetGauges())
	case metrics.Counter:
		t, err := template.New("").Parse(templateCounters)
		if err != nil {
			log.Println(err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		t.Execute(w, storage.GetCounters())
	default:
		err := errors.New("cannot get: no such metrics type <" + metricType + ">")
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
}
