package proxyserver

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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

	router := httpserver.NewRouter(injector, logger)

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

	router.POST("/invoke/:functionName", server.InvokeHandler)

	return server, nil
}

func (s *Server) InvokeHandler(c *gin.Context) {
	functionName := c.Param("functionName")
	invocationUUID := uuid.New()

	s.logger.WithFields(logrus.Fields{
		"requestUUID":  invocationUUID,
		"functionName": functionName,
	}).Info("Invoking...")

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.Error(err)
	}

	subject := fmt.Sprintf("%s.%s", core.InvokeSubjectBase, invocationUUID)

	response, err := s.publisher.PublishWaitReply(c, subject, body)
	if err != nil {
		c.Error(err)
	}

	// TODO: develop protocol.
	_, err = c.Writer.Write(response)
	if err != nil {
		c.Error(err)
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
