package httpserver

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/samber/do"
	"github.com/sirupsen/logrus"
)

type Server struct {
	injector *do.Injector
	server   http.Server
	router   *gin.Engine
	logger   logrus.FieldLogger

	error error
}

func NewServer(injector *do.Injector, name string, port int) (*Server, error) {
	logger, err := do.Invoke[logrus.FieldLogger](injector)
	if err != nil {
		return nil, err
	}

	logger = logger.WithField("component", name)

	defer logger.Info("Server created.")

	router := NewRouter(injector, logger)

	server := &Server{
		injector: injector,
		server: http.Server{
			Addr:              fmt.Sprintf("0.0.0.0:%d", port),
			ReadHeaderTimeout: ReadHeaderTimeout,
			Handler:           router,
		},
		logger: logger,
		router: router,
	}

	return server, nil
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

// Logger returns the logger.
func (s *Server) Logger() logrus.FieldLogger { //nolint:ireturn
	return s.logger
}

func (s *Server) Router() *gin.Engine {
	return s.router
}
