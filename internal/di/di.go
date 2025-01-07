package di

import (
	"fmt"

	"github.com/samber/do"
	"github.com/sirupsen/logrus"
	"github.com/zhulik/fid/internal/backends"
	"github.com/zhulik/fid/internal/config"
	"github.com/zhulik/fid/internal/core"
	"github.com/zhulik/fid/internal/forwarder"
	"github.com/zhulik/fid/internal/gateway"
	"github.com/zhulik/fid/internal/infoserver"
	"github.com/zhulik/fid/internal/pubsub"
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

	forwarder.Register(injector)
	backends.Register(injector)
	gateway.Register(injector)
	pubsub.Register(injector)
	infoserver.Register(injector)

	// TODO: inject config

	return injector
}
