package di

import (
	"fmt"

	"github.com/samber/do"
	"github.com/sirupsen/logrus"
	"github.com/zhulik/fid/internal/backends"
	"github.com/zhulik/fid/internal/config"
	"github.com/zhulik/fid/internal/core"
	"github.com/zhulik/fid/internal/gateway"
	"github.com/zhulik/fid/internal/infoserver"
	"github.com/zhulik/fid/internal/invocation"
	"github.com/zhulik/fid/internal/kv"
	"github.com/zhulik/fid/internal/pubsub"
	"github.com/zhulik/fid/internal/runtimeapi"
	"github.com/zhulik/fid/internal/scaler"
)

func New() *do.Injector {
	injector := do.New()

	do.Provide[logrus.FieldLogger](injector, func(_ *do.Injector) (logrus.FieldLogger, error) {
		cfg, err := do.Invoke[core.Config](injector)
		if err != nil {
			return nil, err
		}

		logger := logrus.New()

		logLevel, err := logrus.ParseLevel(cfg.LogLevel())
		if err != nil {
			return nil, fmt.Errorf("failed parse loglevel: %w", err)
		}

		logger.SetLevel(logLevel)

		return logger, nil
	})

	config.Register(injector)

	runtimeapi.Register(injector)
	backends.Register(injector)
	gateway.Register(injector)
	pubsub.Register(injector)
	kv.Register(injector)
	invocation.Register(injector)
	infoserver.Register(injector)
	scaler.Register(injector)

	return injector
}
