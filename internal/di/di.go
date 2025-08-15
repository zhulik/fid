package di

import (
	"context"
	"time"

	"github.com/zhulik/fid/internal/backends"
	"github.com/zhulik/fid/internal/config"
	"github.com/zhulik/fid/internal/httpserver"
	"github.com/zhulik/fid/internal/invocation"
	"github.com/zhulik/fid/internal/kv"
	"github.com/zhulik/fid/internal/logging"
	"github.com/zhulik/fid/internal/pubsub"
	"github.com/zhulik/pal"
)

const (
	initTimeout        = time.Second * 3
	healthCheckTimeout = time.Second * 3
	shutdownTimeout    = time.Second * 30
)

func InitPal(ctx context.Context, cfg *config.Config, services ...pal.ServiceDef) (*pal.Pal, error) {
	p := pal.New(
		append(services,
			pal.Provide(cfg),
			logging.Provide(),
			pubsub.Provide(),
			kv.Provide(),
			invocation.Provide(),
			backends.Provide(),
			httpserver.Provide(),
		)...,
	).
		InitTimeout(initTimeout).
		HealthCheckTimeout(healthCheckTimeout).
		ShutdownTimeout(shutdownTimeout).
		RunHealthCheckServer(":8081", "/health")

	return p, p.Init(ctx) //nolint:wrapcheck
}
