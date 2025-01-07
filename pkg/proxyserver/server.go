package proxyserver

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/samber/do"
	"github.com/sirupsen/logrus"
	"github.com/zhulik/fid/pkg/core"
	"github.com/zhulik/fid/pkg/httpserver"
)

type Server struct {
	injector *do.Injector
	server   http.Server
	error    error

	publisher core.Publisher
	logger    logrus.FieldLogger
}

// NewServer creates a new Server instance.
func NewServer(injector *do.Injector) (*Server, error) {
	logger, err := do.Invoke[logrus.FieldLogger](injector)
	if err != nil {
		return nil, err
	}

	logger = logger.WithField("component", "proxyserver.Server")

	publisher, err := do.Invoke[core.Publisher](injector)
	if err != nil {
		return nil, err
	}

	defer logger.Info("Server created.")

	router := mux.NewRouter()
	// router.Use(httpserver.JSONMiddleware(logger))
	// router.Use(httpserver.RecoverMiddleware(logger))
	// router.Use(httpserver.LoggingMiddleware(logger))

	config, err := do.Invoke[core.Config](injector)
	if err != nil {
		return nil, err
	}

	server := &Server{
		injector: injector,
		server: http.Server{
			Addr:              fmt.Sprintf("0.0.0.0:%d", config.ProxyServerPort()),
			ReadHeaderTimeout: httpserver.ReadHeaderTimeout,
			Handler:           router,
		},
		logger:    logger,
		publisher: publisher,
	}

	router.HandleFunc("/pulse", server.PulseHandler).Methods("GET").Name("pulse")

	router.HandleFunc("/invoke/{functionName}", server.InvokeHandler).Methods("POST").Name("invoke")

	router.HandleFunc("/", server.NotFoundHandler)

	return server, nil
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

func (s *Server) InvokeHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	vars := mux.Vars(r)
	functionName := vars["functionName"]

	invocationUUID := uuid.New()

	s.logger.WithFields(logrus.Fields{
		"requestUUID":  invocationUUID,
		"functionName": functionName,
	}).Info("Invoking...")

	body, err := io.ReadAll(r.Body)
	if err != nil {
		panic(err)
	}

	subject := fmt.Sprintf("%s.%s", core.InvokeSubjectBase, invocationUUID)

	response, err := s.publisher.PublishWaitReply(r.Context(), subject, body)
	if err != nil {
		panic(err)
	}

	// TODO: develop protocol.
	_, err = w.Write(response)
	if err != nil {
		panic(err)
	}
}

func (s *Server) NotFoundHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	err := httpserver.WriteJSON(httpserver.ErrorBody{
		Error: "Not found",
	}, w, http.StatusNotFound)
	if err != nil {
		panic(err)
	}
}

func (s *Server) HealthCheck() error {
	s.logger.Debug("Server health check.")

	return s.error
}

func (s *Server) Shutdown() error {
	s.logger.Debug("Server shutting down...")
	defer s.logger.Debug("Server shot down.")

	err := s.server.Shutdown(context.Background())
	if err != nil {
		return fmt.Errorf("failed to shut down the https server: %w", err)
	}

	return nil
}

// Run starts the HTTP server.
func (s *Server) Run() error {
	s.logger.Debug("Starting server at: ", s.server.Addr)

	s.error = s.server.ListenAndServe()

	return s.error
}
