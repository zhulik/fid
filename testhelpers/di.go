package testhelpers

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/samber/do/v2"
	"github.com/samber/lo"
	"github.com/zhulik/fid/internal/config"
	"github.com/zhulik/fid/internal/logging"
	pubSubNats "github.com/zhulik/fid/internal/pubsub/nats"
	"github.com/zhulik/pal"
)

//nolint:mnd
func NewPal(ctx context.Context, services ...pal.ServiceDef) *pal.Pal {
	services = append(services,
		logging.Provide(),
		pal.Provide(&config.Config{}),
		pal.Provide(&pubSubNats.Client{}),
	)

	p := pal.New(
		services...,
	).
		InitTimeout(time.Second * 10).
		HealthCheckTimeout(time.Second * 10).
		ShutdownTimeout(time.Second * 10)

	lo.Must0(p.Init(ctx))

	return p
}

func NewInjector() do.Injector {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelWarn,
	}))
	slog.SetDefault(logger)

	injector := do.New()
	do.ProvideValue(injector, logger)

	do.ProvideValue(injector, config.Config{})

	return injector
}
