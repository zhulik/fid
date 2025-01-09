package gateway

import (
	"fmt"
	"io"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/samber/do"
	"github.com/sirupsen/logrus"
	core2 "github.com/zhulik/fid/internal/core"
	"github.com/zhulik/fid/pkg/httpserver"
)

type Server struct {
	*httpserver.Server

	publisher core2.Publisher
}

// NewServer creates a new Server instance.
func NewServer(injector *do.Injector) (*Server, error) {
	config, err := do.Invoke[core2.Config](injector)
	if err != nil {
		return nil, err
	}

	server, err := httpserver.NewServer(injector, "gateway.Server", config.GatewayPort())
	if err != nil {
		return nil, fmt.Errorf("failed to create a new http server: %w", err)
	}

	publisher, err := do.Invoke[core2.Publisher](injector)
	if err != nil {
		return nil, err
	}

	srv := &Server{
		Server:    server,
		publisher: publisher,
	}

	srv.Router.POST("/invoke/:functionName", srv.InvokeHandler)

	return srv, nil
}

func (s *Server) InvokeHandler(c *gin.Context) {
	functionName := c.Param("functionName")
	invocationUUID := uuid.New()

	s.Logger.WithFields(logrus.Fields{
		"requestUUID":  invocationUUID,
		"functionName": functionName,
	}).Info("Invoking...")

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.Error(err)
	}

	subject := fmt.Sprintf("%s.%s", core2.InvokeSubjectBase, invocationUUID)

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
