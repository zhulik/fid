package forwarder

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/samber/do"
	"github.com/zhulik/fid/internal/core"
	"github.com/zhulik/fid/pkg/httpserver"
)

type Server struct {
	*httpserver.Server

	backend    core.ContainerBackend
	subscriber core.Subscriber
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

	subscriber, err := do.Invoke[core.Subscriber](injector)
	if err != nil {
		return nil, err
	}

	server.Router.Use(JWTMiddleware())

	srv := &Server{
		Server:     server,
		backend:    backend,
		subscriber: subscriber,
	}

	// Mimicking the AWS Lambda runtime API for custom runtimes
	srv.Router.GET("/2018-06-01/runtime/invocation/next", srv.NextHandler)
	// TODO:
	// srv.Router.POST("/2018-06-01/runtime/invocation/:requestID/response", srv.ResponseHandler)
	// srv.Router.POST("/2018-06-01/runtime/invocation/:requestID/error", srv.ErrorHandler)
	// srv.Router.POST("/2018-06-01/runtime/init/error", srv.InitErrorHandler)

	return srv, nil
}

func (s *Server) NextHandler(c *gin.Context) {
	functionNameAny, ok := c.Get("functionName")
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "function name not found in the context"})

		return
	}

	functionName, ok := functionNameAny.(string)
	if !ok || functionName == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "function name not found in the context"})

		return
	}

	_, err := s.backend.Function(c.Request.Context(), functionName)
	if err != nil {
		if errors.Is(err, core.ErrFunctionNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "function not found"})

			return
		}

		c.Error(err)

		return
	}

	logger := s.Logger.WithField("function", functionName)

	logger.Debug("Function connected, waiting for events...")

	subject := fmt.Sprintf("%s.%s", core.InvokeSubjectBase, functionName)

	msg, err := s.subscriber.Next(c.Request.Context(), core.InvocationStreamName, functionName, subject)
	if err != nil {
		c.Error(err)

		return
	}

	logger.Infof("Event received: %s", msg.Headers()["Lambda-Runtime-Aws-Request-Id"][0])

	for key, values := range msg.Headers() {
		for _, value := range values {
			c.Writer.Header().Add(key, value)
		}
	}

	_, err = c.Writer.Write(msg.Data())
	if err != nil {
		c.Error(err)

		return
	}
}
