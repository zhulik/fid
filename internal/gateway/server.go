package gateway

import (
	"errors"
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

	invoker, err := do.Invoke[core.Invoker](injector)
	if err != nil {
		return nil, err
	}

	srv := &Server{
		Server:  server,
		invoker: invoker,
	}

	srv.Router.POST("/invoke/:functionName", srv.InvokeHandler)

	return srv, nil
}

func (s *Server) InvokeHandler(c *gin.Context) {
	ctx := c.Request.Context()

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.Error(err)

		return
	}

	reply, err := s.invoker.Invoke(ctx, c.Param("functionName"), body)
	if err != nil {
		if errors.Is(err, core.ErrFunctionNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "function not found"})

			return
		}

		c.Error(err)

		return
	}

	_, err = c.Writer.Write(reply)
	if err != nil {
		c.Error(err)

		return
	}
}
