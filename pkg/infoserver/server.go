package infoserver

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
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

	server, err := httpserver.NewServer(injector, "infoserver.Server", config.InfoServerPort())
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

	srv.Router.GET("/info", srv.InfoHandler)

	return srv, nil
}

func (s *Server) InfoHandler(c *gin.Context) {
	info, err := s.backend.Info(c)
	if err != nil {
		c.Error(err)
	}

	c.IndentedJSON(http.StatusOK, info)
}
