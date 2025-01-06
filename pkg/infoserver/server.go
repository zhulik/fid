package infoserver

import (
	"context"
	"fmt"
	"net/http"

	"github.com/sirupsen/logrus"

	"github.com/gorilla/mux"
	"github.com/samber/do"
	"github.com/zhulik/fid/pkg/core"
	"github.com/zhulik/fid/pkg/httpserver"
)

type Server struct {
	injector *do.Injector
	backend  core.ContainerBackend
	server   http.Server
	error    error

	logger logrus.FieldLogger
}

// NewServer creates a new Server instance.
func NewServer(injector *do.Injector) (*Server, error) {
	logger, err := do.Invoke[logrus.FieldLogger](injector)
	if err != nil {
		return nil, err
	}
	logger = logger.WithField("component", "infoserver.Server")

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

	server := &Server{
		injector: injector,
		backend:  backend,
		server: http.Server{
			Addr:              fmt.Sprintf("0.0.0.0:%d", config.InfoServerPort()),
			Handler:           router,
			ReadHeaderTimeout: httpserver.ReadHeaderTimeout,
		},
		logger: logger,
	}

	router.HandleFunc("/info", server.InfoHandler).Methods("GET").Name("info")
	router.HandleFunc("/pulse", server.PulseHandler).Methods("GET").Name("pulse")

	router.HandleFunc("/", server.NotFoundHandler)

	return server, nil
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

func (s *Server) PulseHandler(_ http.ResponseWriter, r *http.Request) {
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
	s.logger.Info("Server health check.")

	return s.error
}

func (s *Server) Shutdown() error {
	s.logger.Info("Server shutting down...")
	defer s.logger.Info("Server shot down.")

	return s.server.Shutdown(context.Background())
}

// Run starts the HTTP server.
func (s *Server) Run() error {
	s.logger.Info("Starting server at: ", s.server.Addr)

	s.error = s.server.ListenAndServe()

	return s.error
}
