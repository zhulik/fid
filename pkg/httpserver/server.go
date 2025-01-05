package httpserver

import (
	"context"
	"fmt"
	"github.com/samber/do"
	"github.com/zhulik/fid/pkg/core"
	"net/http"

	"github.com/zhulik/fid/pkg/log"
)

var (
	logger = log.Logger.WithField("component", "httpserver.Server")
)

type Server struct {
	injector *do.Injector
	backend  core.Backend
	server   http.Server
	error    error
}

// NewServer creates a new Server instance
func NewServer(injector *do.Injector) (*Server, error) {
	logger.Info("Creating new server...")
	defer logger.Info("Server created.")

	mux := http.NewServeMux()

	backend, err := do.Invoke[core.Backend](injector)
	if err != nil {
		return nil, err
	}

	s := &Server{
		injector: injector,
		backend:  backend,
		server: http.Server{
			Addr:    fmt.Sprintf("0.0.0.0:8080"), // TODO: read port from config
			Handler: mux,
		},
	}

	mux.HandleFunc("/info", Middlewares(s.InfoHandler))

	mux.HandleFunc("/pulse", Middlewares(s.PulseHandler))

	mux.HandleFunc("/", Middlewares(s.NotFoundHandler))

	return s, nil
}

func (s *Server) InfoHandler(w http.ResponseWriter, r *http.Request) {
	info, err := s.backend.Info(r.Context())
	if err != nil {
		panic(err)
	}

	err = WriteJSON(info, w)
	if err != nil {
		panic(err)
	}
}

func (s *Server) PulseHandler(w http.ResponseWriter, r *http.Request) {
	errs := s.injector.HealthCheck()

	for _, err := range errs {
		if err != nil {
			panic(err)
		}
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) NotFoundHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	err := WriteJSON(ErrorBody{
		Error: "Not found",
	}, w)
	if err != nil {
		panic(err)
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
