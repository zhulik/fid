package infoserver

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
	"github.com/zhulik/fid/internal/config"
	"github.com/zhulik/fid/internal/core"
	"github.com/zhulik/fid/pkg/httpserver"
	"github.com/zhulik/pal"
)

type Server struct {
	*httpserver.Server

	Config        *config.Config
	Logger        *slog.Logger
	Backend       core.ContainerBackend
	FunctionsRepo core.FunctionsRepo

	Pal *pal.Pal
}

// NewServer creates a new Server instance.
func (s *Server) Init(ctx context.Context) error {
	s.Router.GET("/backend", s.BackendHandler)
	s.Router.GET("/functions", s.FunctionsHandler)
	s.Router.GET("/functions/:functionName", s.FunctionHandler)

	return nil
}

func (s *Server) BackendHandler(c *gin.Context) {
	info, err := s.Backend.Info(c)
	if err != nil {
		c.Error(err)
	}

	c.IndentedJSON(http.StatusOK, info)
}

func (s *Server) FunctionsHandler(c *gin.Context) {
	functions, err := s.FunctionsRepo.List(c.Request.Context())
	if err != nil {
		c.Error(err)
	}

	fns := lo.Map(functions, func(fn core.FunctionDefinition, _ int) gin.H {
		return serializeFunction(fn)
	})

	c.IndentedJSON(http.StatusOK, fns)
}

func (s *Server) FunctionHandler(c *gin.Context) {
	function, err := s.FunctionsRepo.Get(c.Request.Context(), c.Param("functionName"))
	if err != nil {
		if errors.Is(err, core.ErrFunctionNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "function not found"})

			return
		}
	}

	c.IndentedJSON(http.StatusOK, serializeFunction(function))
}

func serializeFunction(fn core.FunctionDefinition) gin.H {
	return gin.H{
		"name":     fn.Name(),
		"timeout":  fn.Timeout().Seconds(),
		"minScale": fn.ScalingConfig().Min,
		"maxScale": fn.ScalingConfig().Max,
		// TODO: running instances
		// TODO: something else?
	}
}
