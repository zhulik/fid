package httpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/samber/do"
	"net/http"

	"github.com/zhulik/fid/pkg/log"
)

var logger = log.Logger.WithField("component", "httpserver.Server")

type Server struct {
	injector *do.Injector
	server   http.Server

	error error
}

// NewServer creates a new Server instance
func NewServer(injector *do.Injector) *Server {
	mux := http.NewServeMux()

	s := &Server{
		injector: injector,
		server: http.Server{
			Addr:    fmt.Sprintf("0.0.0.0:8080"), // TODO: read port from config
			Handler: mux,
		}}

	mux.HandleFunc("/hello", LoggingMiddleware(RecoverMiddleware(s.HelloHandler)))

	return s
}

func (s *Server) HelloHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	err := json.NewEncoder(w).Encode("test")

	if err != nil {
		logger.WithError(err).Error("failed to encode response")
	}
}

func (s *Server) HealthCheck() error {
	logger.Info("Server health check.")
	return s.error
}

func (s *Server) Shutdown() error {
	logger.Info("Server shutting down...")
	defer logger.Info("Server shot down.")

	return s.server.Shutdown(context.Background())
}

// Run starts the HTTP server
func (s *Server) Run() error {
	logger.Info("Starting server at: ", s.server.Addr)

	s.error = s.server.ListenAndServe()
	return s.error
}
