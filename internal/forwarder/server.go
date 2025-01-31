package forwarder

import (
	"fmt"
	"io"
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
	publisher  core.Publisher
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

	publisher, err := do.Invoke[core.Publisher](injector)
	if err != nil {
		return nil, err
	}

	server.Router.Use(JWTMiddleware())
	server.Router.Use(FunctionMiddleware(backend))

	srv := &Server{
		Server:     server,
		backend:    backend,
		subscriber: subscriber,
		publisher:  publisher,
	}

	// Mimicking the AWS Lambda runtime API for custom runtimes
	srv.Router.GET("/2018-06-01/runtime/invocation/next", srv.NextHandler)
	srv.Router.POST("/2018-06-01/runtime/invocation/:requestID/response", srv.ResponseHandler)
	srv.Router.POST("/2018-06-01/runtime/invocation/:requestID/error", srv.ErrorHandler)
	srv.Router.POST("/2018-06-01/runtime/init/error", srv.InitErrorHandler)

	return srv, nil
}

func (s *Server) NextHandler(c *gin.Context) {
	function := c.MustGet("function").(core.Function) //nolint:forcetypeassert
	logger := s.Logger.WithField("function", function.Name())

	logger.Debug("Function connected, waiting for events...")

	subject := fmt.Sprintf("%s.%s", core.InvokeSubjectBase, function.Name())

	msg, err := s.subscriber.Next(c.Request.Context(), core.InvocationStreamName, function.Name(), subject)
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

	c.Data(http.StatusOK, "application/octet-stream", msg.Data())
}

func (s *Server) ResponseHandler(c *gin.Context) {
	requestID := c.Param("requestID")

	logger := s.Logger.WithField("requestID", requestID)

	logger.Debug("Sending reply...")

	reply, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.Error(err)

		return
	}

	msg := core.Msg{
		Subject: fmt.Sprintf("%s.%s", core.ReplySubjectBase, requestID),
		Data:    gin.H{"reply": reply},
	}

	if err := s.publisher.Publish(c.Request.Context(), msg); err != nil {
		c.Error(err)

		return
	}

	logger.Info("Reply sent")
}

func (s *Server) ErrorHandler(c *gin.Context) {
	// TODO: implement
	requestID := c.Param("requestID")

	logger := s.Logger.WithField("requestID", requestID)

	logger.Debug("Sending error reply...")

	reply, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.Error(err)

		return
	}

	msg := core.Msg{
		Subject: fmt.Sprintf("%s.%s", core.ReplySubjectBase, requestID),
		Data:    gin.H{"error": reply},
	}

	if err := s.publisher.Publish(c.Request.Context(), msg); err != nil {
		c.Error(err)

		return
	}

	logger.Info("Error reply sent")
}

func (s *Server) InitErrorHandler(_ *gin.Context) {
	// TODO: implement
	panic("not implemented")
}
