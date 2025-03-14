package logging

import (
	"context"
	"fmt"

	"github.com/samber/do/v2"
	"github.com/sirupsen/logrus"
	"github.com/zhulik/fid/internal/core"
)

func Register(ctx context.Context, injector do.Injector) {
	do.Provide(injector, func(injector do.Injector) (logrus.FieldLogger, error) {
		cfg := do.MustInvoke[core.Config](injector)
		logger := logrus.New()

		logLevel, err := logrus.ParseLevel(cfg.LogLevel())
		if err != nil {
			return nil, fmt.Errorf("failed parse loglevel: %w", err)
		}

		logger.SetLevel(logLevel)

		return logger, nil
	})
}
