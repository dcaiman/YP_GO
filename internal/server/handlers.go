package server

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"text/template"

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

func (srv *ServerConfig) handlerCheckDBConnection(w http.ResponseWriter, r *http.Request) {
	if err := srv.Storage.AccessCheck(r.Context()); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_, err := w.Write([]byte("STORAGE IS AVAILABLE"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (srv *ServerConfig) handlerUpdateBatch(w http.ResponseWriter, r *http.Request) {
	batch := []metric.Metric{}
	for i := range batch {
		if err := srv.Storage.UpdateMetricFromStruct(batch[i]); err != nil {
			log.Println(err.Error())
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}
}

func (srv *ServerConfig) handlerUpdateJSON(w http.ResponseWriter, r *http.Request) {
	content, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	m := metric.Metric{}
	if err := m.SetFromJSON(content); err != nil {
		log.Println(err.Error())
		return
	}

	if r.Header.Get("Hash") != "" && srv.Cfg.HashKey != "" {
		resHash, err := srv.checkHash(m)
		w.Header().Set("Hash", resHash)
		if err != nil {
			log.Println(err.Error())
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}

	if err := checkTypeSupport(m.MType); err != nil {
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusNotImplemented)
		return
	}

	exists, err := srv.Storage.MetricExists(m.ID)
	if err != nil {
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if exists {
		switch m.MType {
		case Gauge:
			err = srv.Storage.UpdateValue(m.ID, *m.Value)
		case Counter:
			err = srv.Storage.AddDelta(m.ID, *m.Delta)
		}
		if err != nil {
			log.Println(err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		if err := srv.Storage.UpdateMetricFromStruct(m); err != nil {
			log.Println(err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	if srv.Cfg.SyncUpload {
		srv.Storage.UploadStorage()
	}
}

func (srv *ServerConfig) handlerUpdateDirect(w http.ResponseWriter, r *http.Request) {
	mType := chi.URLParam(r, "type")
	err := checkTypeSupport(mType)
	if err != nil {
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusNotImplemented)
		return
	}

	mVal := chi.URLParam(r, "val")
	var mValue float64
	var mDelta int64
	switch mType {
	case Gauge:
		mValue, err = strconv.ParseFloat(mVal, 64)
	case Counter:
		mDelta, err = strconv.ParseInt(mVal, 10, 64)
	}
	if err != nil {
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	mName := chi.URLParam(r, "name")
	exists, err := srv.Storage.MetricExists(mName)
	if err != nil {
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if exists {
		switch mType {
		case Gauge:
			err = srv.Storage.UpdateValue(mName, mValue)
		case Counter:
			fmt.Println(mDelta)
			err = srv.Storage.AddDelta(mName, mDelta)
		}
		if err != nil {
			log.Println(err.Error())
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	} else {
		m := metric.Metric{
			ID:    mName,
			MType: mType,
			Value: &mValue,
			Delta: &mDelta,
		}
		m.UpdateHash(srv.Cfg.HashKey)
		if err = srv.Storage.NewMetric(m); err != nil {
			log.Println(err.Error())
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}

	if srv.Cfg.SyncUpload {
		srv.Storage.UploadStorage()
	}
}

func (srv *ServerConfig) handlerGetAll(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	t, err := template.New("").Parse(templateHandlerGetAll)
	if err != nil {
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	allMetrics, err := srv.Storage.GetAllMetrics()
	if err != nil {
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	t.Execute(w, allMetrics)
}

func (srv *ServerConfig) handlerGetMetricJSON(w http.ResponseWriter, r *http.Request) {
	content, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	mReq := metric.Metric{}
	if err := mReq.SetFromJSON(content); err != nil {
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := checkTypeSupport(mReq.MType); err != nil {
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusNotImplemented)
		return
	}

	mRes, err := srv.Storage.GetMetric(mReq.ID)
	if err != nil {
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if mRes.MType != mReq.MType {
		http.Error(w, "cannot get: metric <"+mReq.ID+"> is not <"+mReq.MType+">", http.StatusNotFound)
		return
	}

	tmp := mRes.Hash //

	if err := mRes.UpdateHash(srv.Cfg.HashKey); err != nil {
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if tmp != mReq.Hash { //
		fmt.Println()
		fmt.Println("STRANGE!!!")
		log.Println(tmp, "\n", mRes.Hash)
		fmt.Println(mRes)
		fmt.Println()
	}

	mResJSON, err := mRes.GetJSON()
	if err != nil {
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", JSONCT)
	w.Write(mResJSON)
}

func (srv *ServerConfig) handlerGetMetric(w http.ResponseWriter, r *http.Request) {
	mType := chi.URLParam(r, "type")
	if err := checkTypeSupport(mType); err != nil {
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusNotImplemented)
		return
	}

	mName := chi.URLParam(r, "name")
	m, err := srv.Storage.GetMetric(mName)
	if err != nil {
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if m.MType != mType {
		http.Error(w, "cannot get: metric <"+mName+"> is not <"+mType+">", http.StatusNotFound)
		return
	}

	switch m.MType {
	case Gauge:
		_, err = w.Write([]byte(strconv.FormatFloat(*m.Value, 'f', 3, 64)))
	case Counter:
		_, err = w.Write([]byte(strconv.FormatInt(*m.Delta, 10)))
	}
	if err != nil {
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
	err := errors.New("unsupported type <" + mType + ">")
	return err
}

func (srv *ServerConfig) checkHash(m metric.Metric) (string, error) {
	h := m.Hash
	m.UpdateHash(srv.Cfg.HashKey)
	if h != m.Hash {
		err := errors.New("inconsistent hashes")
		return m.Hash, err
	}
	return m.Hash, nil
}
