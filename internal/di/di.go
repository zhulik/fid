package di

import (
	"context"
	"fmt"

	"github.com/samber/do"
	"github.com/sirupsen/logrus"
	"github.com/zhulik/fid/internal/backends"
	"github.com/zhulik/fid/internal/core"
	"github.com/zhulik/fid/internal/gateway"
	"github.com/zhulik/fid/internal/infoserver"
	"github.com/zhulik/fid/internal/invocation"
	"github.com/zhulik/fid/internal/kv"
	"github.com/zhulik/fid/internal/pubsub"
	"github.com/zhulik/fid/internal/runtimeapi"
	"github.com/zhulik/fid/internal/scaler"
)

func Init() *do.Injector {
	injector := do.New()

	do.Provide(injector, func(injector *do.Injector) (logrus.FieldLogger, error) {
		cfg := do.MustInvoke[core.Config](injector)
		logger := logrus.New()

		logLevel, err := logrus.ParseLevel(cfg.LogLevel())
		if err != nil {
			return nil, fmt.Errorf("failed parse loglevel: %w", err)
		}

		logger.SetLevel(logLevel)

		return logger, nil
	})

	ctx := context.Background()

	runtimeapi.Register(ctx, injector)
	backends.Register(ctx, injector)
	gateway.Register(ctx, injector)
	pubsub.Register(ctx, injector)
	kv.Register(ctx, injector)
	invocation.Register(ctx, injector)
	infoserver.Register(ctx, injector)
	scaler.Register(ctx, injector)

	return injector
}
