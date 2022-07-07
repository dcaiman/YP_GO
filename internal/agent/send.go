package agent

import (
	"bytes"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"

	"github.com/dcaiman/YP_GO/internal/clog"
)

func (agn *AgentConfig) sendMetric(name string) error {
	var url, val string
	var body []byte

	m, err := agn.Storage.GetMetric(name)
	if err != nil {
		return clog.ToLog(clog.FuncName(), err)
	}

	switch agn.Cfg.CType {
	case TextPlainCT:
		switch m.MType {
		case Gauge:
			val = strconv.FormatFloat(*m.Value, 'f', 3, 64)
		case Counter:
			val = strconv.FormatInt(*m.Delta, 10)
		default:
			return clog.ToLog(clog.FuncName(), errors.New("cannot send: unsupported metric type <"+m.MType+">"))
		}
		url = agn.Cfg.SrvAddr + "/update/" + m.MType + "/" + m.ID + "/" + val
		body = nil
	case JSONCT:
		tmpBody, err := m.GetJSON()
		if err != nil {
			return clog.ToLog(clog.FuncName(), err)
		}
		url = agn.Cfg.SrvAddr + "/update/"
		body = tmpBody
	default:
		return clog.ToLog(clog.FuncName(), errors.New("cannot send: unsupported content type <"+agn.Cfg.CType+">"))
	}
	res, err := customPostRequest(HTTPStr+url, agn.Cfg.CType, m.Hash, bytes.NewBuffer(body))
	if err != nil {
		return clog.ToLog(clog.FuncName(), err)
	}
	defer res.Body.Close()
	if m.MType == Counter {
		if err := agn.Storage.ResetDelta(name); err != nil {
			return clog.ToLog(clog.FuncName(), err)
		}
	}
	log.Println("SEND METRIC: ", res.Status, res.Request.URL)
	return nil
}

func (agn *AgentConfig) sendBatch() error {
	body, err := agn.getStorageBatch(true)
	if err != nil {
		return clog.ToLog(clog.FuncName(), err)
	}
	res, err := customPostRequest(HTTPStr+agn.Cfg.SrvAddr+"/updates/", JSONCT, "", bytes.NewBuffer(body))
	if err != nil {
		return clog.ToLog(clog.FuncName(), err)
	}
	defer res.Body.Close()
	log.Println("SEND BATCH: ", res.Status, res.Request.URL)
	return nil
}

func customPostRequest(url, contentType, hash string, body io.Reader) (resp *http.Response, err error) {
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, clog.ToLog(clog.FuncName(), err)
	}
	if hash != "" {
		req.Header.Set("Hash", hash)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	client, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, clog.ToLog(clog.FuncName(), err)
	}
	return client, nil
}
