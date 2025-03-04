package runtimeapi

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/samber/do"
	"github.com/sirupsen/logrus"
	"github.com/zhulik/fid/internal/core"
	"github.com/zhulik/fid/pkg/httpserver"
)

type Server struct {
	*httpserver.Server

	pubSuber         core.PubSuber
	functionInstance functionInstance
}

// NewServer creates a new Server instance.
func NewServer(injector *do.Injector) (*Server, error) {
	config := do.MustInvoke[core.Config](injector)

	if config.FunctionName() == "" {
		return nil, core.ErrFunctionNameNotGiven
	}

	logger := do.MustInvoke[logrus.FieldLogger](injector).WithFields(map[string]interface{}{
		"component": "runtimeapi.Server",
		"function":  config.FunctionName(),
	})
	functionsRepo := do.MustInvoke[core.FunctionsRepo](injector)
	pubSuber := do.MustInvoke[core.PubSuber](injector)
	instancesRepo := do.MustInvoke[core.InstancesRepo](injector)

	server, err := httpserver.NewServer(injector, logger, config.HTTPPort())
	if err != nil {
		return nil, fmt.Errorf("failed to create a new http server: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second) //nolint:mnd
	defer cancel()

	function, err := functionsRepo.Get(ctx, config.FunctionName())
	if err != nil {
		return nil, fmt.Errorf("failed to get function: %w", err)
	}

	instance := functionInstance{
		FunctionDefinition: function,
		id:                 config.FunctionInstanceID(),
		instancesRepo:      instancesRepo,
	}

	err = instance.add(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to add function to instances repo: %w", err)
	}

	srv := &Server{
		Server:           server,
		pubSuber:         pubSuber,
		functionInstance: instance,
	}

	// Mimicking the AWS Lambda runtime API for custom runtimes
	srv.Router.GET("/2018-06-01/runtime/invocation/next", srv.NextHandler)
	srv.Router.POST("/2018-06-01/runtime/invocation/:requestID/response", srv.ResponseHandler)
	srv.Router.POST("/2018-06-01/runtime/invocation/:requestID/error", srv.ErrorHandler)
	srv.Router.POST("/2018-06-01/runtime/init/error", srv.InitErrorHandler)

	return srv, nil
}

func (s *Server) Shutdown() error {
	err := s.Server.Shutdown()
	if err != nil {
		return err //nolint:wrapcheck
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	return s.functionInstance.delete(ctx)
}

func (s *Server) NextHandler(c *gin.Context) {
	ctx := c.Request.Context()
	subject := s.pubSuber.ConsumeSubjectName(s.functionInstance)

	s.Logger.Info("Function connected, waiting for events...")

	streamName := s.pubSuber.FunctionStreamName(s.functionInstance)

	err := s.functionInstance.busy(c.Request.Context(), false)
	if err != nil {
		c.Error(err)

		return
	}

	msg, err := s.pubSuber.Next(ctx, streamName, []string{subject}, s.functionInstance.Name())
	if err != nil {
		c.Error(err)

		return
	}

	msg.Ack()

	s.Logger.Infof("Event received: %s", msg.Headers()[core.HeaderNameRequestID][0])

	for key, values := range msg.Headers() {
		for _, value := range values {
			c.Writer.Header().Add(key, value)
		}
	}

	c.Data(http.StatusOK, core.ContentTypeJSON, msg.Data())
}

func (s *Server) ResponseHandler(c *gin.Context) {
	requestID := c.Param("requestID")
	subject := s.pubSuber.ResponseSubjectName(s.functionInstance, requestID)

	logger := s.Logger.WithFields(map[string]interface{}{
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

	err = s.functionInstance.executed(c.Request.Context())
	if err != nil {
		c.Error(err)

		return
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
		"function":  function,
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

	err = s.functionInstance.executed(c.Request.Context())
	if err != nil {
		c.Error(err)

		return
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
