package testhelpers

import (
	"context"
	"log/slog"
	"os"

	"github.com/samber/do/v2"
	"github.com/samber/lo"
	"github.com/zhulik/fid/internal/config"
	"github.com/zhulik/fid/internal/logging"
	natsPubSub "github.com/zhulik/fid/internal/pubsub/nats"
	"github.com/zhulik/pal"
)

func NewPal(ctx context.Context) *pal.Pal {
	p := pal.New(
		logging.Provide(),
		pal.Provide(&config.Config{}),
		pal.Provide(&natsPubSub.Client{}),
	)

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
	do.Provide(injector, natsPubSub.NewClient)

	return injector
}
