package agent

import (
	"bytes"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"
)

func (agn *AgentConfig) sendMetric(name string) error {
	var url, val string
	var body []byte

	m, err := agn.Storage.GetMetric(name)
	if err != nil {
		log.Println(err.Error())
		return err
	}

	switch agn.Cfg.CType {
	case TextPlainCT:
		switch m.MType {
		case Gauge:
			val = strconv.FormatFloat(*m.Value, 'f', 3, 64)
		case Counter:
			val = strconv.FormatInt(*m.Delta, 10)
		default:
			err := errors.New("cannot send: unsupported metric type <" + m.MType + ">")
			log.Println(err)
			return err
		}
		url = agn.Cfg.SrvAddr + "/update/" + m.MType + "/" + m.ID + "/" + val
		body = nil
	case JSONCT:
		tmpBody, err := m.GetJSON()
		if err != nil {
			log.Println(err.Error())
			return err
		}
		url = agn.Cfg.SrvAddr + "/update/"
		body = tmpBody
	default:
		err := errors.New("cannot send: unsupported content type <" + agn.Cfg.CType + ">")
		log.Println(err)
		return err
	}
	res, err := customPostRequest(HTTPStr+url, agn.Cfg.CType, m.Hash, bytes.NewBuffer(body))
	if err != nil {
		log.Println(err.Error())
		return err
	}
	defer res.Body.Close()
	if m.MType == Counter {
		agn.Storage.ResetDelta(name)
	}
	log.Println(res.Status, res.Request.URL)
	return nil
}

func (agn *AgentConfig) sendBatch() error {
	body, err := agn.getStorageBatch(true)
	if err != nil {
		return err
	}
	res, err := customPostRequest(HTTPStr+agn.Cfg.SrvAddr+"/updates/", JSONCT, "", bytes.NewBuffer(body))
	if err != nil {
		log.Println(err.Error())
		return err
	}
	defer res.Body.Close()
	log.Println("SEND BATCH: ", res.Status, res.Request.URL)
	return nil
}

func customPostRequest(url, contentType, hash string, body io.Reader) (resp *http.Response, err error) {
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}
	if hash != "" {
		req.Header.Set("Hash", hash)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	return http.DefaultClient.Do(req)
}
