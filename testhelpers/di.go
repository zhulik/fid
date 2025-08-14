package testhelpers

import (
	"log/slog"
	"os"

	"github.com/samber/do/v2"
	"github.com/zhulik/fid/internal/config"
	natsPubSub "github.com/zhulik/fid/internal/pubsub/nats"
)

func NewInjector() do.Injector {
	injector := do.New()
	do.ProvideValue[*slog.Logger](injector, slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelWarn,
	})))

	do.ProvideValue(injector, config.Config{})
	do.Provide(injector, natsPubSub.NewClient)

	return injector
}
