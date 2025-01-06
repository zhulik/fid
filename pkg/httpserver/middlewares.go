package httpserver

import (
	"bufio"
	"encoding/json"
	"net"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/sirupsen/logrus"
)

var Middlewares = func(next http.Handler) http.Handler {
	return LoggingMiddleware(RecoverMiddleware(JSONMiddleware(next)))
}

type ResponseWriterWrapper struct {
	http.ResponseWriter
	StatusCode int
}

func (rw *ResponseWriterWrapper) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	// TODO: proper check
	return rw.ResponseWriter.(http.Hijacker).Hijack()
}

func (rw *ResponseWriterWrapper) WriteHeader(code int) {
	rw.StatusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func WriteJSON(doc any, w http.ResponseWriter) error {
	jsonErr, err := json.MarshalIndent(doc, "", " ")

	if err != nil {
		return err
	}

	_, err = w.Write(jsonErr)
	return err
}

// JSONMiddleware sets Content-Type header to "application/json"
func JSONMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}

// LoggingMiddleware logs each request's URI and method
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

		next.ServeHTTP(wrappedWriter, r)
	})
}

// RecoverMiddleware recovers from panics
func RecoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				logger.WithFields(logrus.Fields{
					"error": err,
				}).Error("Application panicked:", string(debug.Stack()))

				w.WriteHeader(http.StatusInternalServerError)
				err := WriteJSON(ErrorBody{
					Error: "Internal Server Error",
				}, w)
				if err != nil {
					logger.WithError(err).Error("cannot write response")
				}
			}
		}()

		next.ServeHTTP(w, r)
	})
}
