package httpserver

import (
	"github.com/zhulik/fid/pkg/log"
	"net/http"
	"time"
)

type ResponseWriterWrapper struct {
	http.ResponseWriter
	StatusCode int
}

func (rw *ResponseWriterWrapper) WriteHeader(code int) {
	rw.StatusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// LoggingMiddleware logs each request's URI and method
func LoggingMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		wrappedWriter := &ResponseWriterWrapper{ResponseWriter: w, StatusCode: http.StatusOK} // Default to 200

		start := time.Now()

		defer func() {
			total := time.Now().Sub(start)
			log.Infof("[%v] %s %s [%s]", wrappedWriter.StatusCode, r.Method, r.URL.Path, total)
		}()

		next(wrappedWriter, r)
	}
}
