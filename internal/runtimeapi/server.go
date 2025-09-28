package runtimeapi

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/nats-io/nats.go"
	"github.com/zhulik/fid/internal/config"
	"github.com/zhulik/fid/internal/core"
	"github.com/zhulik/fid/internal/httpserver"
	"github.com/zhulik/pal"
)

type Server struct {
	*httpserver.Server

	Config        *config.Config
	Logger        *slog.Logger
	PubSuber      core.PubSuber
	FunctionsRepo core.FunctionsRepo
	InstancesRepo core.InstancesRepo
	Pal           *pal.Pal

	functionInstance functionInstance
}

// NewServer creates a new Server instance.
func (s *Server) Init(ctx context.Context) error {
	if s.Config.FunctionName == "" {
		return core.ErrFunctionNameNotGiven
	}

	s.Logger = s.Logger.With(
		"function", s.Config.FunctionName,
	)

	function, err := s.FunctionsRepo.Get(ctx, s.Config.FunctionName)
	if err != nil {
		return fmt.Errorf("failed to get function: %w", err)
	}

	instance := functionInstance{
		FunctionDefinition: function,
		id:                 s.Config.FunctionInstanceID,
		instancesRepo:      s.InstancesRepo,
	}

	err = instance.add(ctx)
	if err != nil {
		return fmt.Errorf("failed to add function to instances repo: %w", err)
	}

	// Mimicking the AWS Lambda runtime API for custom runtimes
	s.Router.GET("/2018-06-01/runtime/invocation/next", s.NextHandler)
	s.Router.POST("/2018-06-01/runtime/invocation/:requestID/response", s.ResponseHandler)
	s.Router.POST("/2018-06-01/runtime/invocation/:requestID/error", s.ErrorHandler)
	s.Router.POST("/2018-06-01/runtime/init/error", s.InitErrorHandler)

	return nil
}

func (s *Server) Run(ctx context.Context) error {
	return s.RunServer(ctx) //nolint:wrapcheck
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.functionInstance.delete(ctx)
}

func (s *Server) NextHandler(c *gin.Context) {
	ctx := c.Request.Context()
	subject := s.PubSuber.InvokeSubjectName(s.functionInstance)

	s.Logger.Info("Function connected, waiting for events...")

	streamName := s.PubSuber.FunctionStreamName(s.functionInstance)

	err := s.functionInstance.busy(c.Request.Context(), false)
	if err != nil {
		c.Error(err)

		return
	}

	msg, err := s.PubSuber.Next(ctx, streamName, []string{subject}, s.functionInstance.Name())
	if err != nil {
		c.Error(err)

		return
	}

	msg.Ack() //nolint:errcheck

	s.Logger.Info("Event received", "requestID", msg.Headers()[core.HeaderNameRequestID][0])

	for key, values := range msg.Headers() {
		for _, value := range values {
			c.Writer.Header().Add(key, value)
		}
	}

	c.Data(http.StatusOK, core.ContentTypeJSON, msg.Data())
}

func (s *Server) ResponseHandler(c *gin.Context) {
	requestID := c.Param("requestID")
	subject := s.PubSuber.ResponseSubjectName(s.functionInstance, requestID)

	logger := s.Logger.With(
		"requestID", requestID,
		"subject", subject,
	)

	logger.Info("Sending response...")

	response, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.Error(err)

		return
	}

	msg := nats.NewMsg(subject)
	msg.Data = response

	err = s.functionInstance.executed(c.Request.Context())
	if err != nil {
		c.Error(err)

		return
	}

	if err := s.PubSuber.Publish(c.Request.Context(), msg); err != nil {
		c.Error(err)

		return
	}

	logger.Debug("Response sent")
}

func (s *Server) ErrorHandler(c *gin.Context) {
	requestID := c.Param("requestID")
	subject := s.PubSuber.ErrorSubjectName(s.functionInstance, requestID)

	logger := s.Logger.With(
		"requestID", requestID,
		"subject", subject,
	)

	logger.Info("Sending error response...")

	response, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.Error(err)

		return
	}

	msg := nats.NewMsg(subject)
	msg.Data = response

	if err := s.PubSuber.Publish(c.Request.Context(), msg); err != nil {
		c.Error(err)

		return
	}

	logger.Info("Error response sent")
}

func (s *Server) InitErrorHandler(_ *gin.Context) {
	// TODO: implement
	panic("not implemented")
}
