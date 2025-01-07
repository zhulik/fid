package infoserver

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/samber/do"
	"github.com/sirupsen/logrus"
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

	defer logger.Info("Server created.")

	router := httpserver.NewRouter(logger)

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

	router.GET("/info", server.InfoHandler)
	router.GET("/pulse", server.PulseHandler)

	return server, nil
}

func (s *Server) InfoHandler(c *gin.Context) {
	info, err := s.backend.Info(c)
	if err != nil {
		c.Error(err)
	}

	c.JSON(http.StatusOK, info)
}

func (s *Server) PulseHandler(c *gin.Context) {
	errs := s.injector.HealthCheck()

	for _, err := range errs {
		if err != nil {
			c.Error(err)
		}
	}
}

func (s *Server) HealthCheck() error {
	s.logger.Debug("Server health check.")

	return s.error
}

func (s *Server) Shutdown() error {
	s.logger.Info("Server shutting down...")
	defer s.logger.Info("Server shot down.")

	err := s.server.Shutdown(context.Background())
	if err != nil {
		return fmt.Errorf("failed to shut down the https server: %w", err)
	}

	return nil
}

// Run starts the HTTP server.
func (s *Server) Run() error {
	s.logger.Info("Starting server at: ", s.server.Addr)

	s.error = s.server.ListenAndServe()

	return s.error
}
