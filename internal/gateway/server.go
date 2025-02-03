package gateway

import (
	"context"
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

	invoker core.Invoker
}

// NewServer creates a new Server instance.
func NewServer(injector *do.Injector) (*Server, error) {
	config, err := do.Invoke[core.Config](injector)
	if err != nil {
		return nil, err
	}

	server, err := httpserver.NewServer(injector, "gateway.Server", config.HTTPPort())
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

	server.Router.Use(middlewares.FunctionMiddleware(backend, func(c *gin.Context) string {
		return c.Param("functionName")
	}))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server.Logger.Debug("Creating or updating function streams.")

	functions, err := backend.Functions(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get functions: %w", err)
	}

	for _, function := range functions {
		err := invoker.CreateOrUpdateFunctionStream(ctx, function)
		if err != nil {
			return nil, fmt.Errorf("failed to create or update function stream: %w", err)
		}
	}

	// TODO: delete streams for deleted functions.

	srv := &Server{
		Server:  server,
		invoker: invoker,
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
