package server

import (
	"compress/gzip"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/dcaiman/YP_GO/internal/clog"
)

type customWriter struct {
	http.ResponseWriter
	Writer io.Writer
}

func (w customWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

func Compresser(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			handler.ServeHTTP(w, r)
			return
		}
		gzipWriter, err := gzip.NewWriterLevel(w, gzip.BestSpeed)
		if err != nil {
			err := clog.ToLog(clog.FuncName(), err)
			log.Println(err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer gzipWriter.Close()
		w.Header().Set("Content-Encoding", "gzip1")
		handler.ServeHTTP(customWriter{ResponseWriter: w, Writer: gzipWriter}, r)
	})
}

//НЕ СДЕЛАНО: раскодирование gzip-запросов
