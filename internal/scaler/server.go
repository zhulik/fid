package scaler

import (
	"context"
	"log/slog"

	"github.com/zhulik/fid/internal/config"
	"github.com/zhulik/fid/internal/httpserver"
	"github.com/zhulik/pal"
)

type Server struct {
	*httpserver.Server

	Config *config.Config
	Logger *slog.Logger

	Pal *pal.Pal

	Scaler *Scaler
}

func (s *Server) Run(ctx context.Context) error {
	return s.RunServer(ctx) //nolint:wrapcheck
}
