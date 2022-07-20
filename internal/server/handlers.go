package server

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"text/template"

	"github.com/dcaiman/YP_GO/internal/clog"
	"github.com/dcaiman/YP_GO/internal/metric"
	"github.com/go-chi/chi/v5"
)

const (
	templateHandlerGetAll = "METRICS LIST: <p>{{range .}}{{.ID}}: {{.Value}}{{.Delta}} ({{.MType}})</p>{{end}}"
)

var supportedTypes = [...]string{
	Gauge,
	Counter,
}

const (
	Gauge       = "gauge"
	Counter     = "counter"
	TextPlainCT = "text/plain"
	JSONCT      = "application/json"
	HTTPStr     = "http://"
)

func (srv *ServerEnv) handlerCheckDBConnection(w http.ResponseWriter, r *http.Request) {
	if err := srv.Storage.AccessCheck(r.Context()); err != nil {
		err := clog.ToLog(clog.FuncName(), err)
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_, err := w.Write([]byte("STORAGE IS AVAILABLE"))
	if err != nil {
		err := clog.ToLog(clog.FuncName(), err)
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (srv *ServerEnv) handlerUpdateBatch(w http.ResponseWriter, r *http.Request) {
	batch := []metric.Metric{}
	scanner := bufio.NewScanner(r.Body)
	scanner.Scan()
	if err := json.Unmarshal(scanner.Bytes(), &batch); err != nil {
		err := clog.ToLog(clog.FuncName(), err)
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	for i := range batch {
		if _, err := srv.checkHash(batch[i]); err != nil {
			err := clog.ToLog(clog.FuncName(), err)
			log.Println(err.Error())
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		batch[i].Hash = ""
	}
	if err := srv.Storage.UpdateBatch(batch); err != nil {
		err := clog.ToLog(clog.FuncName(), err)
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if srv.Cfg.SyncUpload != nil {
		var tmp struct{}
		srv.Cfg.SyncUpload <- tmp
	}
}

func (srv *ServerEnv) handlerUpdateJSON(w http.ResponseWriter, r *http.Request) {
	mj, err := io.ReadAll(r.Body)
	if err != nil {
		err := clog.ToLog(clog.FuncName(), err)
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	m := metric.Metric{}
	if err := json.Unmarshal(mj, &m); err != nil {
		err := clog.ToLog(clog.FuncName(), err)
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if r.Header.Get("Hash") != "" && srv.Cfg.HashKey != "" {
		resHash, err := srv.checkHash(m)
		w.Header().Set("Hash", resHash)
		if err != nil {
			err := clog.ToLog(clog.FuncName(), err)
			log.Println(err.Error())
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}
	m.Hash = ""

	if err := checkTypeSupport(m.MType); err != nil {
		err := clog.ToLog(clog.FuncName(), err)
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusNotImplemented)
		return
	}

	if err := srv.Storage.UpdateMetric(m); err != nil {
		err := clog.ToLog(clog.FuncName(), err)
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if srv.Cfg.SyncUpload != nil {
		var tmp struct{}
		srv.Cfg.SyncUpload <- tmp
	}
}

func (srv *ServerEnv) handlerUpdateDirect(w http.ResponseWriter, r *http.Request) {
	mType := chi.URLParam(r, "type")
	if err := checkTypeSupport(mType); err != nil {
		err := clog.ToLog(clog.FuncName(), err)
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusNotImplemented)
		return
	}

	mVal := chi.URLParam(r, "val")
	var mValue float64
	var mDelta int64
	var err error
	switch mType {
	case Gauge:
		mValue, err = strconv.ParseFloat(mVal, 64)
	case Counter:
		mDelta, err = strconv.ParseInt(mVal, 10, 64)
	}
	if err != nil {
		err := clog.ToLog(clog.FuncName(), err)
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	mName := chi.URLParam(r, "name")
	m := metric.Metric{
		ID:    mName,
		MType: mType,
		Value: &mValue,
		Delta: &mDelta,
	}
	if err := srv.Storage.UpdateMetric(m); err != nil {
		err := clog.ToLog(clog.FuncName(), err)
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if srv.Cfg.SyncUpload != nil {
		var tmp struct{}
		srv.Cfg.SyncUpload <- tmp
	}
}

func (srv *ServerEnv) handlerGetAll(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	t, err := template.New("").Parse(templateHandlerGetAll)
	if err != nil {
		err := clog.ToLog(clog.FuncName(), err)
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	allMetrics, err := srv.Storage.GetBatch()
	if err != nil {
		err := clog.ToLog(clog.FuncName(), err)
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	t.Execute(w, allMetrics)
}

func (srv *ServerEnv) handlerGetMetricJSON(w http.ResponseWriter, r *http.Request) {
	mjReq, err := io.ReadAll(r.Body)
	if err != nil {
		err := clog.ToLog(clog.FuncName(), err)
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	mReq := metric.Metric{}
	if err := json.Unmarshal(mjReq, &mReq); err != nil {
		err := clog.ToLog(clog.FuncName(), err)
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := checkTypeSupport(mReq.MType); err != nil {
		err := clog.ToLog(clog.FuncName(), err)
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusNotImplemented)
		return
	}

	mRes, err := srv.Storage.GetMetric(mReq.ID)
	if err != nil {
		err := clog.ToLog(clog.FuncName(), err)
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if mRes.MType != mReq.MType {
		http.Error(w, "cannot get: metric <"+mReq.ID+"> is not <"+mReq.MType+">", http.StatusNotFound)
		return
	}

	if err := mRes.UpdateHash(srv.Cfg.HashKey); err != nil {
		err := clog.ToLog(clog.FuncName(), err)
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	mjRes, err := json.Marshal(mRes)
	if err != nil {
		err := clog.ToLog(clog.FuncName(), err)
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", JSONCT)
	w.Write(mjRes)
}

func (srv *ServerEnv) handlerGetMetric(w http.ResponseWriter, r *http.Request) {
	mType := chi.URLParam(r, "type")
	if err := checkTypeSupport(mType); err != nil {
		err := clog.ToLog(clog.FuncName(), err)
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusNotImplemented)
		return
	}

	mName := chi.URLParam(r, "name")
	m, err := srv.Storage.GetMetric(mName)
	if err != nil {
		err := clog.ToLog(clog.FuncName(), err)
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if m.MType != mType {
		err := clog.ToLog(clog.FuncName(), errors.New("cannot get: metric <"+mName+"> is not <"+mType+">"))
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	switch m.MType {
	case Gauge:
		_, err = w.Write([]byte(strconv.FormatFloat(*m.Value, 'f', 3, 64)))
	case Counter:
		_, err = w.Write([]byte(strconv.FormatInt(*m.Delta, 10)))
	}
	if err != nil {
		err := clog.ToLog(clog.FuncName(), err)
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func checkTypeSupport(mType string) error {
	for i := range supportedTypes {
		if mType == supportedTypes[i] {
			return nil
		}
	}
	return clog.ToLog(clog.FuncName(), errors.New("unsupported type <"+mType+">"))
}

func (srv *ServerEnv) checkHash(m metric.Metric) (string, error) {
	tmp := m
	h := m.Hash
	m.UpdateHash(srv.Cfg.HashKey)
	if h != m.Hash {
		fmt.Println(m, tmp)
		return m.Hash, clog.ToLog(clog.FuncName(), errors.New("inconsistent hashes"))
	}
	return m.Hash, nil
}
