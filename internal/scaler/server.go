package scaler

import (
	"log/slog"

	"github.com/zhulik/fid/internal/config"
	"github.com/zhulik/fid/pkg/httpserver"
	"github.com/zhulik/pal"
)

type Server struct {
	*httpserver.Server

	Config *config.Config
	Logger *slog.Logger

	Pal *pal.Pal

	Scaler *Scaler
}
