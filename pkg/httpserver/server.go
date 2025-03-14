package httpserver

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/samber/do/v2"
	"github.com/sirupsen/logrus"
)

type Server struct {
	server http.Server

	Router *gin.Engine
	Logger logrus.FieldLogger

	error error
}

func NewServer(injector do.Injector, logger logrus.FieldLogger, port int) (*Server, error) {
	defer logger.Info("Server created.")

	router := gin.New()

	router.Use(JSONRecovery())
	router.Use(LoggingMiddleware(logger))
	router.Use(JSONErrorHandler(logger))

	router.GET("/health", func(c *gin.Context) {
		errs := injector.HealthCheck()

		for _, err := range errs {
			if err != nil {
				c.Error(err)
			}
		}
	})

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
	s.Logger.Info("Starting server at: ", s.server.Addr)

	s.error = s.server.ListenAndServe()

	return s.error
}
