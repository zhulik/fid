package httpserver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/zhulik/fid/internal/config"
	"github.com/zhulik/pal"
)

type Server struct {
	Config *config.Config

	Pal *pal.Pal

	Router *gin.Engine
	Logger *slog.Logger

	server http.Server
}

func (s *Server) Init(_ context.Context) error {
	defer s.Logger.Info("Server created.")

	router := gin.New()

	router.Use(JSONRecovery())
	router.Use(LoggingMiddleware(s.Logger))
	router.Use(JSONErrorHandler(s.Logger))

	router.GET("/health", func(c *gin.Context) { //nolint:contextcheck
		err := s.Pal.HealthCheck(c.Request.Context())
		if err != nil {
			c.Error(err)
		}
	})

	s.Router = router
	s.server = http.Server{
		Addr:              fmt.Sprintf("0.0.0.0:%d", s.Config.HTTPPort),
		ReadHeaderTimeout: ReadHeaderTimeout,
		Handler:           router,
	}

	return nil
}

// Run starts the HTTP server.
func (s *Server) Run(ctx context.Context) error {
	s.Logger.Info("Starting server", "addr", s.server.Addr)

	go func() {
		<-ctx.Done()
		s.server.Shutdown(ctx) //nolint:errcheck
	}()

	err := s.server.ListenAndServe()
	if !errors.Is(err, http.ErrServerClosed) {
		return err //nolint:wrapcheck
	}

	return nil
}
