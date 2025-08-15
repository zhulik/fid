package di

import (
	"context"
	"fmt"
	"time"

	"github.com/zhulik/fid/internal/backends"
	"github.com/zhulik/fid/internal/config"
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
	services = append(services,
		pal.Provide(cfg),
		logging.Provide(),
		pubsub.Provide(),
		kv.Provide(),
		invocation.Provide(),
		backends.Provide(),
	)
	p := pal.New(
		services...,
	).
		InitTimeout(initTimeout).
		HealthCheckTimeout(healthCheckTimeout).
		ShutdownTimeout(shutdownTimeout)

	err := p.Init(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize pal: %w", err)
	}

	return p, nil
}
