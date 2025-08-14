package logging

import (
	"context"
	"log/slog"
	"os"

	"github.com/samber/do/v2"
	"github.com/zhulik/fid/internal/config"
)

func Register(ctx context.Context, injector do.Injector) {
	do.Provide(injector, func(injector do.Injector) (*slog.Logger, error) {
		cfg := do.MustInvoke[config.Config](injector)
		logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: cfg.LogLevel,
		}))

		return logger, nil
	})
}
