package server

import (
	"compress/gzip"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/dcaiman/YP_GO/internal/clog"
)

var typesToCompress = [...]string{
	"application/javascript",
	"application/json",
	"text/css",
	"text/html",
	"text/plain",
	"text/xml",
}

type customWriter struct {
	http.ResponseWriter
	Writer io.Writer
}

func (w customWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

func Compresser(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Encoding") == "gzip" {
			gzipReader, err := gzip.NewReader(r.Body)
			if err != nil {
				err := clog.ToLog(clog.FuncName(), err)
				log.Println(err.Error())
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			defer gzipReader.Close()
			r.Body = gzipReader
		}

		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") || !contains(typesToCompress[:], r.Header.Get("Content-Type")) {
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
		w.Header().Set("Content-Encoding", "gzip")
		handler.ServeHTTP(customWriter{ResponseWriter: w, Writer: gzipWriter}, r)
	})
}

func contains(arr []string, s string) bool {
	for i := range arr {
		if arr[i] == s {
			return true
		}
	}
	return false
}
