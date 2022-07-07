package server

import (
	"compress/gzip"
	"io"
	"log"
	"net/http"
	"strings"
)

type customWriter struct {
	http.ResponseWriter
	Writer io.Writer
}

func (w customWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

func Compresser(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			handler.ServeHTTP(w, r)
			return
		}
		gzipWriter, err := gzip.NewWriterLevel(w, gzip.BestSpeed)
		if err != nil {
			log.Println(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer gzipWriter.Close()
		w.Header().Set("Content-Encoding", "gzip")
		handler.ServeHTTP(customWriter{ResponseWriter: w, Writer: gzipWriter}, r)
	}
}

//НЕ СДЕЛАНО: раскодирование gzip-запросов
