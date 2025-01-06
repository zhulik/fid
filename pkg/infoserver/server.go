package infoserver

import (
	"context"
	"fmt"
	"github.com/zhulik/fid/pkg/httpserver"
	"net/http"

	"github.com/samber/do"

	"github.com/gorilla/mux"

	"github.com/zhulik/fid/pkg/core"
	"github.com/zhulik/fid/pkg/log"
)

var (
	logger = log.Logger.WithField("component", "infoserver.Server")
)

type Server struct {
	injector *do.Injector
	backend  core.ContainerBackend
	server   http.Server
	error    error
}

// NewServer creates a new Server instance
func NewServer(injector *do.Injector) (*Server, error) {
	logger.Info("Creating new server...")
	defer logger.Info("Server created.")

	router := mux.NewRouter()
	router.Use(httpserver.JSONMiddleware(logger))
	router.Use(httpserver.RecoverMiddleware(logger))
	router.Use(httpserver.LoggingMiddleware(logger))

	config, err := do.Invoke[core.Config](injector)
	if err != nil {
		return nil, err
	}

	backend, err := do.Invoke[core.ContainerBackend](injector)
	if err != nil {
		return nil, err
	}

	s := &Server{
		injector: injector,
		backend:  backend,
		server: http.Server{
			Addr:    fmt.Sprintf(fmt.Sprintf("0.0.0.0:%d", config.InfoServerPort())),
			Handler: router,
		},
	}

	router.HandleFunc("/info", s.InfoHandler).Methods("GET").Name("info")
	router.HandleFunc("/pulse", s.PulseHandler).Methods("GET").Name("pulse")

	router.HandleFunc("/", s.NotFoundHandler)

	return s, nil
}

func (s *Server) InfoHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	info, err := s.backend.Info(r.Context())
	if err != nil {
		panic(err)
	}

	err = httpserver.WriteJSON(info, w, http.StatusOK)
	if err != nil {
		panic(err)
	}
}

func (s *Server) PulseHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	errs := s.injector.HealthCheck()

	for _, err := range errs {
		if err != nil {
			panic(err)
		}
	}
}

func (s *Server) NotFoundHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	w.WriteHeader(http.StatusNotFound)
	err := httpserver.WriteJSON(httpserver.ErrorBody{
		Error: "Not found",
	}, w, http.StatusOK)
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
