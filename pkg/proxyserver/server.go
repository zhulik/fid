package proxyserver

import (
	"fmt"
	"io"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/samber/do"
	"github.com/sirupsen/logrus"
	"github.com/zhulik/fid/pkg/core"
	"github.com/zhulik/fid/pkg/httpserver"
)

type Server struct {
	*httpserver.Server

	publisher core.Publisher
}

// NewServer creates a new Server instance.
func NewServer(injector *do.Injector) (*Server, error) {
	config, err := do.Invoke[core.Config](injector)
	if err != nil {
		return nil, err
	}

	server, err := httpserver.NewServer(injector, "proxyserver.Server", config.ProxyServerPort())
	if err != nil {
		return nil, fmt.Errorf("failed to create a new http server: %w", err)
	}

	logger := server.Logger()

	publisher, err := do.Invoke[core.Publisher](injector)
	if err != nil {
		return nil, err
	}

	defer logger.Info("Server created.")

	srv := &Server{
		Server:    server,
		publisher: publisher,
	}

	server.Router().POST("/invoke/:functionName", srv.InvokeHandler)

	return srv, nil
}

func (s *Server) InvokeHandler(c *gin.Context) {
	functionName := c.Param("functionName")
	invocationUUID := uuid.New()

	s.Logger().WithFields(logrus.Fields{
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
