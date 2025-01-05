package httpserver

import (
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
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
			logger.WithFields(logrus.Fields{
				"method":   r.Method,
				"path":     r.URL.Path,
				"duration": total,
				"status":   wrappedWriter.StatusCode,
			}).Infof("%s %s", r.Method, r.URL.Path)
		}()

		next(wrappedWriter, r)
	}
}

// RecoverMiddleware recovers from panics
func RecoverMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				logger.Error("recovered from panic: ", err)
				w.WriteHeader(http.StatusInternalServerError)
			}
		}()

		next(w, r)
	}
}
