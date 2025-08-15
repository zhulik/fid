package logging

import (
	"context"
	"log/slog"
	"os"

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
