package gateway

import (
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/samber/do"
	"github.com/sirupsen/logrus"
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
	config := do.MustInvoke[core.Config](injector)
	logger := do.MustInvoke[logrus.FieldLogger](injector).WithField("component", "gateway.Server")
	functionsRepo := do.MustInvoke[core.FunctionsRepo](injector)
	invoker := do.MustInvoke[core.Invoker](injector)

	server, err := httpserver.NewServer(injector, logger, config.HTTPPort())
	if err != nil {
		return nil, fmt.Errorf("failed to create a new http server: %w", err)
	}

	server.Router.Use(middlewares.FunctionMiddleware(functionsRepo, func(c *gin.Context) string {
		return c.Param("functionName")
	}))

	server.Logger.Debug("Creating or updating function streams.")

	srv := &Server{
		Server:  server,
		invoker: invoker,
	}

	srv.Router.POST("/invoke/:functionName", srv.InvokeHandler)

	return srv, nil
}

func (s *Server) InvokeHandler(c *gin.Context) {
	ctx := c.Request.Context()

	function := c.MustGet("function").(core.FunctionDefinition) //nolint:forcetypeassert

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
