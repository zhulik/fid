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

func init() { //nolint:gochecknoinits
	do.Provide[logrus.FieldLogger](nil, func(_ *do.Injector) (logrus.FieldLogger, error) {
		cfg := do.MustInvoke[core.Config](nil)
		logger := logrus.New()

		logLevel, err := logrus.ParseLevel(cfg.LogLevel())
		if err != nil {
			return nil, fmt.Errorf("failed parse loglevel: %w", err)
		}

		logger.SetLevel(logLevel)

		return logger, nil
	})

	config.Register()

	runtimeapi.Register()
	backends.Register()
	gateway.Register()
	pubsub.Register()
	kv.Register()
	invocation.Register()
	infoserver.Register()
	scaler.Register()
}

func Logger() logrus.FieldLogger {
	return do.MustInvoke[logrus.FieldLogger](nil)
}
