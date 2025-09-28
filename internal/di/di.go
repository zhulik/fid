package di

import (
	"context"
	"time"

	"github.com/zhulik/fid/internal/backends"
	"github.com/zhulik/fid/internal/config"
	"github.com/zhulik/fid/internal/httpserver"
	"github.com/zhulik/fid/internal/invocation"
	"github.com/zhulik/fid/internal/kv"
	"github.com/zhulik/fid/internal/pubsub"
	"github.com/zhulik/pal"
)

const (
	initTimeout        = time.Second * 3
	healthCheckTimeout = time.Second * 3
	shutdownTimeout    = time.Second * 30
)

func Run(ctx context.Context, cfg *config.Config, services ...pal.ServiceDef) error {
	services = append(services,
		pal.Provide(cfg),
		pubsub.Provide(),
		kv.Provide(),
		invocation.Provide(),
		backends.Provide(),
		httpserver.Provide(),
	)

	p := pal.New(services...).
		InjectSlog().
		InitTimeout(initTimeout).
		HealthCheckTimeout(healthCheckTimeout).
		ShutdownTimeout(shutdownTimeout).
		RunHealthCheckServer(":8081", "/health")

	return p.Run(ctx) //nolint:wrapcheck
}
