package forwarder

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/samber/do"
	"github.com/zhulik/fid/internal/core"
	"github.com/zhulik/fid/internal/middlewares"
	"github.com/zhulik/fid/pkg/httpserver"
)

var ErrFunctionNameNotGiven = errors.New("function name is not provided as env FUNCTION_NAME")

type Server struct {
	*httpserver.Server

	backend  core.ContainerBackend
	pubSuber core.PubSuber
}

// NewServer creates a new Server instance.
func NewServer(injector *do.Injector) (*Server, error) {
	config, err := do.Invoke[core.Config](injector)
	if err != nil {
		return nil, err
	}

	server, err := httpserver.NewServer(injector, "forwarder.Server", config.GatewayPort())
	if err != nil {
		return nil, fmt.Errorf("failed to create a new http server: %w", err)
	}

	backend, err := do.Invoke[core.ContainerBackend](injector)
	if err != nil {
		return nil, err
	}

	pubSuber, err := do.Invoke[core.PubSuber](injector)
	if err != nil {
		return nil, err
	}

	functionName := os.Getenv("FUNCTION_NAME")

	// TODO: move to config
	if functionName == "" {
		return nil, ErrFunctionNameNotGiven
	}

	server.Router.Use(middlewares.FunctionMiddleware(backend, func(c *gin.Context) string {
		return functionName
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
	function := c.MustGet("function").(core.Function) //nolint:forcetypeassert
	subject := s.pubSuber.InvokeSubjectName(function.Name())

	logger := s.Logger.WithField("function", function.Name())

	logger.Debug("Function connected, waiting for events...")

	streamName := s.pubSuber.FunctionStreamName(function.Name())

	msg, err := s.pubSuber.Next(ctx, streamName, []string{subject}, function.Name())
	if err != nil {
		c.Error(err)

		return
	}

	err = msg.Ack()
	if err != nil {
		c.Error(err)

		return
	}

	logger.Infof("Event received: %s", msg.Headers()[core.RequestIDHeaderName][0])

	for key, values := range msg.Headers() {
		for _, value := range values {
			c.Writer.Header().Add(key, value)
		}
	}

	c.Data(http.StatusOK, core.ContentTypeJSON, msg.Data())
}

func (s *Server) ResponseHandler(c *gin.Context) {
	requestID := c.Param("requestID")
	function := c.MustGet("function").(core.Function) //nolint:forcetypeassert
	subject := s.pubSuber.ResponseSubjectName(function.Name(), requestID)

	logger := s.Logger.WithFields(map[string]interface{}{
		"function":  function.Name(),
		"requestID": requestID,
		"subject":   subject,
	})

	logger.Debug("Sending response...")

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
	function := c.MustGet("function").(core.Function) //nolint:forcetypeassert
	subject := s.pubSuber.ErrorSubjectName(function.Name(), requestID)

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
		Subject: s.pubSuber.ErrorSubjectName(function.Name(), requestID),
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
