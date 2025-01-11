package gateway

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/samber/do"
	"github.com/sirupsen/logrus"
	"github.com/zhulik/fid/internal/core"
	"github.com/zhulik/fid/pkg/httpserver"
)

type Server struct {
	*httpserver.Server

	backend   core.ContainerBackend
	publisher core.Publisher
}

// NewServer creates a new Server instance.
func NewServer(injector *do.Injector) (*Server, error) {
	config, err := do.Invoke[core.Config](injector)
	if err != nil {
		return nil, err
	}

	server, err := httpserver.NewServer(injector, "gateway.Server", config.GatewayPort())
	if err != nil {
		return nil, fmt.Errorf("failed to create a new http server: %w", err)
	}

	publisher, err := do.Invoke[core.Publisher](injector)
	if err != nil {
		return nil, err
	}

	backend, err := do.Invoke[core.ContainerBackend](injector)
	if err != nil {
		return nil, err
	}

	srv := &Server{
		Server:    server,
		publisher: publisher,
		backend:   backend,
	}

	srv.Router.POST("/invoke/:functionName", srv.InvokeHandler)

	return srv, nil
}

func (s *Server) InvokeHandler(c *gin.Context) {
	ctx := c.Request.Context()

	functionName := c.Param("functionName")

	function, err := s.backend.Function(ctx, functionName)
	if err != nil {
		if errors.Is(err, core.ErrFunctionNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "function not found"})

			return
		}

		c.Error(err)

		return
	}

	invocationUUID := uuid.New()

	s.Logger.WithFields(logrus.Fields{
		"requestUUID":  invocationUUID,
		"functionName": functionName,
	}).Info("Invoking...")

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.Error(err)

		return
	}

	subject := fmt.Sprintf("%s.%s.%s", core.InvokeSubjectBase, functionName, invocationUUID)

	response, err := s.publisher.PublishWaitReply(ctx, subject, body, function.Timeout())
	if err != nil {
		if errors.Is(err, context.Canceled) {
			s.Logger.Info("client disconnected while waiting for reply")

			return
		}

		c.Error(err)

		return
	}

	// TODO: develop protocol.
	_, err = c.Writer.Write(response)
	if err != nil {
		c.Error(err)

		return
	}
}
