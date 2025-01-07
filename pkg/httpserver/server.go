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
	server http.Server

	Router *gin.Engine
	Logger logrus.FieldLogger

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
		server: http.Server{
			Addr:              fmt.Sprintf("0.0.0.0:%d", port),
			ReadHeaderTimeout: ReadHeaderTimeout,
			Handler:           router,
		},
		Logger: logger,
		Router: router,
	}

	return server, nil
}

func (s *Server) HealthCheck() error {
	s.Logger.Debug("Server health check.")

	return s.error
}

func (s *Server) Shutdown() error {
	s.Logger.Debug("Server shutting down...")
	defer s.Logger.Debug("Server shot down.")

	err := s.server.Shutdown(context.Background())
	if err != nil {
		return fmt.Errorf("failed to shut down the https server: %w", err)
	}

	return nil
}

// Run starts the HTTP server.
func (s *Server) Run() error {
	s.Logger.Debug("Starting server at: ", s.server.Addr)

	s.error = s.server.ListenAndServe()

	return s.error
}
