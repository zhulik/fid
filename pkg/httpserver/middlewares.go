package httpserver

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

const (
	ReadHeaderTimeout = 5 * time.Second
)

var ErrNotImplementsHijacker = errors.New("ResponseWriter does not implement http.Hijacker")

type ResponseWriterWrapper struct {
	http.ResponseWriter
	StatusCode int
}

func (rw *ResponseWriterWrapper) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := rw.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, ErrNotImplementsHijacker
	}

	return hijacker.Hijack() //nolint:wrapcheck
}

func (rw *ResponseWriterWrapper) WriteHeader(code int) {
	rw.StatusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func WriteJSON(doc any, w http.ResponseWriter, status int) error {
	jsonErr, err := json.MarshalIndent(doc, "", " ")
	if err != nil {
		return fmt.Errorf("failed marshal json: %w", err)
	}

	w.WriteHeader(status)

	_, err = w.Write(jsonErr)
	if err != nil {
		return fmt.Errorf("failed to write response: %w", err)
	}

	return nil
}

// JSONMiddleware sets Content-Type header to "application/json".
func JSONMiddleware(_ logrus.FieldLogger) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			next.ServeHTTP(w, r)
		})
	}
}

// LoggingMiddleware logs each request's URI and method.
func LoggingMiddleware(logger logrus.FieldLogger) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			wrappedWriter := &ResponseWriterWrapper{ResponseWriter: w, StatusCode: http.StatusOK} // Default to 200

			start := time.Now()

			defer func() {
				total := time.Since(start)
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
}

// RecoverMiddleware recovers from panics.
func RecoverMiddleware(logger logrus.FieldLogger) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					logger.WithFields(logrus.Fields{
						"error": err,
					}).Error("Application panicked:", string(debug.Stack()))

					err := WriteJSON(ErrorBody{
						Error: "Internal Server Error",
					}, w, http.StatusInternalServerError)
					if err != nil {
						logger.WithError(err).Error("cannot write response")
					}
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}
