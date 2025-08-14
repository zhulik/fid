package logging

import (
	"context"
	"log/slog"
	"os"

	"github.com/samber/do/v2"
	"github.com/zhulik/fid/internal/config"
	"github.com/zhulik/pal"
)

func Provide() pal.ServiceDef {
	return pal.ProvideList(
		pal.ProvideFn(func(ctx context.Context) (*slog.Logger, error) {
			p := pal.FromContext(ctx)

			cfg, err := pal.Invoke[*config.Config](ctx, p)
			if err != nil {
				return nil, err
			}

			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
				Level: cfg.LogLevel,
			}))

			return logger, nil
		}),
	)
}

func Register(ctx context.Context, injector do.Injector) {
	do.Provide(injector, func(injector do.Injector) (*slog.Logger, error) {
		cfg := do.MustInvoke[config.Config](injector)
		logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: cfg.LogLevel,
		}))
		slog.SetDefault(logger)

		return logger, nil
	})
}
