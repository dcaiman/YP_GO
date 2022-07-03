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

const (
	templateHandlerGetAll = "METRICS LIST: <p>{{range .Metrics}}{{.ID}}: {{.Value}}{{.Delta}} ({{.MType}})<p>{{end}}"
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

func (srv *ServerConfig) handlerUpdateJSON(w http.ResponseWriter, r *http.Request) {
	content, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	m, err := metric.SetFromJSON(&metric.Metric{}, content)
	if err != nil {
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

	if srv.Storage.MetricExists(m.ID, m.MType) {
		switch m.MType {
		case Gauge:
			err = srv.Storage.UpdateValue(m.ID, srv.Cfg.HashKey, *m.Value)
		case Counter:
			err = srv.Storage.AddDelta(m.ID, srv.Cfg.HashKey, *m.Delta)
		}
		if err != nil {
			log.Println(err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		srv.Storage.UpdateMetricFromStruct(m)
		return
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
	if srv.Storage.MetricExists(mName, mType) {
		switch mType {
		case Gauge:
			err = srv.Storage.UpdateValue(mName, srv.Cfg.HashKey, mValue)
		case Counter:
			fmt.Println(mDelta)
			err = srv.Storage.AddDelta(mName, srv.Cfg.HashKey, mDelta)
		}
		if err != nil {
			log.Println(err.Error())
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	} else {
		err = srv.Storage.NewMetric(mName, mType, srv.Cfg.HashKey, &mValue, &mDelta)
		if err != nil {
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
		log.Println("cannot get: " + err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	t.Execute(w, &srv.Storage)
}

func (srv *ServerConfig) handlerGetMetricJSON(w http.ResponseWriter, r *http.Request) {
	content, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	mReq, err := metric.SetFromJSON(&metric.Metric{}, content)
	if err != nil {
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
	if err != nil || mRes.MType != mReq.MType {
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusNotFound)
		return
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

	m, err := srv.Storage.GetMetric(chi.URLParam(r, "name"))
	if err != nil || m.MType != mType {
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
