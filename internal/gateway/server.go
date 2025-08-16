package gateway

import (
	"context"
	"io"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/zhulik/fid/internal/config"
	"github.com/zhulik/fid/internal/core"
	"github.com/zhulik/fid/internal/httpserver"
	"github.com/zhulik/fid/internal/middlewares"
	"github.com/zhulik/pal"
)

type Server struct {
	*httpserver.Server

	Config        *config.Config
	Logger        *slog.Logger
	FunctionsRepo core.FunctionsRepo
	Invoker       core.Invoker

	Pal *pal.Pal
}

// NewServer creates a new Server instance.
func (s *Server) Init(ctx context.Context) error {
	s.Router.Use(middlewares.FunctionMiddleware(s.FunctionsRepo, func(c *gin.Context) string {
		return c.Param("functionName")
	}))

	s.Router.POST("/invoke/:functionName", s.InvokeHandler)

	return nil
}

func (s *Server) Run(ctx context.Context) error {
	return s.RunServer(ctx) //nolint:wrapcheck
}

func (s *Server) InvokeHandler(c *gin.Context) {
	ctx := c.Request.Context()

	function := c.MustGet("function").(core.FunctionDefinition) //nolint:forcetypeassert

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.Error(err)

		return
	}

	response, err := s.Invoker.Invoke(ctx, function, body)
	if err != nil {
		c.Error(err)

		return
	}

	c.Data(http.StatusOK, core.ContentTypeJSON, response)
}
