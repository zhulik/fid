package gateway

import (
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/samber/do"
	"github.com/zhulik/fid/internal/core"
	"github.com/zhulik/fid/internal/middlewares"
	"github.com/zhulik/fid/pkg/httpserver"
)

type Server struct {
	*httpserver.Server

	backend core.ContainerBackend

	invoker core.Invoker
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

	backend, err := do.Invoke[core.ContainerBackend](injector)
	if err != nil {
		return nil, err
	}

	invoker, err := do.Invoke[core.Invoker](injector)
	if err != nil {
		return nil, err
	}

	server.Router.Use(middlewares.FunctionMiddleware(backend))

	srv := &Server{
		Server:  server,
		invoker: invoker,
		backend: backend,
	}

	srv.Router.POST("/invoke/:functionName", srv.InvokeHandler)

	return srv, nil
}

func (s *Server) InvokeHandler(c *gin.Context) {
	ctx := c.Request.Context()

	function := c.MustGet("function").(core.Function) //nolint:forcetypeassert

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.Error(err)

		return
	}

	response, err := s.invoker.Invoke(ctx, function, body)
	if err != nil {
		c.Error(err)

		return
	}

	c.Data(http.StatusOK, core.ContentTypeJSON, response)
}
