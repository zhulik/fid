package wsserver

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/samber/do"
	"github.com/zhulik/fid/pkg/core"
	"github.com/zhulik/fid/pkg/httpserver"
)

type Server struct {
	*httpserver.Server

	backend core.ContainerBackend
}

// NewServer creates a new Server instance.
func NewServer(injector *do.Injector) (*Server, error) {
	config, err := do.Invoke[core.Config](injector)
	if err != nil {
		return nil, err
	}

	server, err := httpserver.NewServer(injector, "wsserver.Server", config.WSServerPort())
	if err != nil {
		return nil, fmt.Errorf("failed to create a new http server: %w", err)
	}

	backend, err := do.Invoke[core.ContainerBackend](injector)
	if err != nil {
		return nil, err
	}

	srv := &Server{
		Server:  server,
		backend: backend,
	}

	// TODO: authentication
	srv.Router.GET("/ws/:functionName", srv.WebsocketHandler)

	return srv, nil
}

func (s *Server) WebsocketHandler(c *gin.Context) {
	functionName := c.Param("functionName")
	// _, err := s.backend.Function(r.Context(), functionName)
	// if errors.Is(err, core.ErrFunctionNotFound) {
	//	err := WriteJSON(ErrorBody{Error: "Not found"}, w)
	//	if err != nil {
	//		panic(err)
	//	}
	//	return
	//}

	s.Logger.Debug("Function '", functionName, "' handler connected, upgrading to websocket connection...")
	// Upgrade the HTTP connection to a WebSocket connection
	upgrader := websocket.Upgrader{
		CheckOrigin: func(_ *http.Request) bool { return true },
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.Error(err)
	}

	s.Logger.Debug("Function '", functionName, "' handler successfully upgraded to websocket connection")

	wsConn := NewWebsocketConnection(functionName, conn, s.Logger)
	// TODO: handle error?
	go wsConn.Handle() //nolint:errcheck
}
