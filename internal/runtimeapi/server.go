package runtimeapi

import (
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/samber/do"
	"github.com/sirupsen/logrus"
	"github.com/zhulik/fid/internal/core"
	"github.com/zhulik/fid/internal/middlewares"
	"github.com/zhulik/fid/pkg/httpserver"
)

type Server struct {
	*httpserver.Server

	backend  core.ContainerBackend
	pubSuber core.PubSuber
}

// NewServer creates a new Server instance.
func NewServer(injector *do.Injector) (*Server, error) {
	config := do.MustInvoke[core.Config](injector)

	if config.FunctionName() == "" {
		return nil, core.ErrFunctionNameNotGiven
	}

	logger := do.MustInvoke[logrus.FieldLogger](injector).WithFields(map[string]interface{}{
		"component":    "runtimeapi.Server",
		"functionName": config.FunctionName(),
	})
	backend := do.MustInvoke[core.ContainerBackend](injector)
	pubSuber := do.MustInvoke[core.PubSuber](injector)

	server, err := httpserver.NewServer(injector, logger, config.HTTPPort())
	if err != nil {
		return nil, fmt.Errorf("failed to create a new http server: %w", err)
	}

	server.Router.Use(middlewares.FunctionMiddleware(backend, func(c *gin.Context) string {
		return config.FunctionName()
	}))

	srv := &Server{
		Server:   server,
		backend:  backend,
		pubSuber: pubSuber,
	}

	// Mimicking the AWS Lambda runtime API for custom runtimes
	srv.Router.GET("/2018-06-01/runtime/invocation/next", srv.NextHandler)
	srv.Router.POST("/2018-06-01/runtime/invocation/:requestID/response", srv.ResponseHandler)
	srv.Router.POST("/2018-06-01/runtime/invocation/:requestID/error", srv.ErrorHandler)
	srv.Router.POST("/2018-06-01/runtime/init/error", srv.InitErrorHandler)

	return srv, nil
}

func (s *Server) NextHandler(c *gin.Context) {
	ctx := c.Request.Context()
	function := c.MustGet("function").(core.FunctionDefinition) //nolint:forcetypeassert
	subject := s.pubSuber.ConsumeSubjectName(function)

	logger := s.Logger.WithField("function", function.Name())

	logger.Info("Function connected, waiting for events...")

	streamName := s.pubSuber.FunctionStreamName(function)

	msg, err := s.pubSuber.Next(ctx, streamName, []string{subject}, function.Name())
	if err != nil {
		c.Error(err)

		return
	}

	msg.Ack()

	logger.Infof("Event received: %s", msg.Headers()[core.HeaderNameRequestID][0])

	for key, values := range msg.Headers() {
		for _, value := range values {
			c.Writer.Header().Add(key, value)
		}
	}

	c.Data(http.StatusOK, core.ContentTypeJSON, msg.Data())
}

func (s *Server) ResponseHandler(c *gin.Context) {
	requestID := c.Param("requestID")
	function := c.MustGet("function").(core.FunctionDefinition) //nolint:forcetypeassert
	subject := s.pubSuber.ResponseSubjectName(function, requestID)

	logger := s.Logger.WithFields(map[string]interface{}{
		"function":  function.Name(),
		"requestID": requestID,
		"subject":   subject,
	})

	logger.Info("Sending response...")

	response, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.Error(err)

		return
	}

	msg := core.Msg{
		Subject: subject,
		Data:    response,
	}

	if err := s.pubSuber.Publish(c.Request.Context(), msg); err != nil {
		c.Error(err)

		return
	}

	logger.Debug("Response sent")
}

func (s *Server) ErrorHandler(c *gin.Context) {
	requestID := c.Param("requestID")
	function := c.MustGet("function").(core.FunctionDefinition) //nolint:forcetypeassert
	subject := s.pubSuber.ErrorSubjectName(function, requestID)

	logger := s.Logger.WithFields(map[string]interface{}{
		"function":  function.Name(),
		"requestID": requestID,
		"subject":   subject,
	})

	logger.Debug("Sending error response...")

	response, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.Error(err)

		return
	}

	msg := core.Msg{
		Subject: s.pubSuber.ErrorSubjectName(function, requestID),
		Data:    response,
	}

	if err := s.pubSuber.Publish(c.Request.Context(), msg); err != nil {
		c.Error(err)

		return
	}

	logger.Info("Error response sent")
}

func (s *Server) InitErrorHandler(_ *gin.Context) {
	// TODO: implement
	panic("not implemented")
}
