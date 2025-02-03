package infoserver

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/samber/do"
	"github.com/samber/lo"
	"github.com/zhulik/fid/internal/core"
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

	server, err := httpserver.NewServer(injector, "infoserver.Server", config.HTTPPort())
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

	srv.Router.GET("/backend", srv.BackendHandler)
	srv.Router.GET("/functions", srv.FunctionsHandler)
	srv.Router.GET("/functions/:functionName", srv.FunctionHandler)

	return srv, nil
}

func (s *Server) BackendHandler(c *gin.Context) {
	info, err := s.backend.Info(c)
	if err != nil {
		c.Error(err)
	}

	c.IndentedJSON(http.StatusOK, info)
}

func (s *Server) FunctionsHandler(c *gin.Context) {
	functions, err := s.backend.Functions(c.Request.Context())
	if err != nil {
		c.Error(err)
	}

	fns := lo.Map(functions, func(fn core.Function, _ int) gin.H {
		return serializeFunction(fn)
	})

	c.IndentedJSON(http.StatusOK, fns)
}

func (s *Server) FunctionHandler(c *gin.Context) {
	function, err := s.backend.Function(c.Request.Context(), c.Param("functionName"))
	if err != nil {
		if errors.Is(err, core.ErrFunctionNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "function not found"})

			return
		}
	}

	c.IndentedJSON(http.StatusOK, serializeFunction(function))
}

func serializeFunction(fn core.Function) gin.H {
	return gin.H{
		"name":     fn.Name(),
		"timeout":  fn.Timeout().Seconds(),
		"minScale": fn.ScalingConfig().Min,
		"maxScale": fn.ScalingConfig().Max,
		// TODO: running instances
		// TODO: something else?
	}
}
