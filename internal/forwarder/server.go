package forwarder

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
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

	srv := &Server{
		Server:     server,
		backend:    backend,
		subscriber: subscriber,
	}

	// TODO: authentication
	srv.Router.GET("/ws/:functionName", srv.WebsocketHandler)

	return srv, nil
}

func (s *Server) WebsocketHandler(c *gin.Context) {
	functionName := c.Param("functionName")

	_, err := s.backend.Function(c.Request.Context(), functionName)
	if err != nil {
		if errors.Is(err, core.ErrFunctionNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "function not found"})

			return
		}

		c.Error(err)
	}

	s.Logger.WithField("function", functionName).Debug("Function connected, upgrading to websocket connection...")
	// Upgrade the HTTP connection to a WebSocket connection
	upgrader := websocket.Upgrader{
		CheckOrigin: func(_ *http.Request) bool { return true },
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.Error(err)
	}

	s.Logger.WithField("function", functionName).Debug("Function successfully upgraded to websocket connection")

	wsConn := NewWebsocketConnection(functionName, conn, s.Logger)
	// TODO: handle error?
	go wsConn.Handle() //nolint:errcheck
}
