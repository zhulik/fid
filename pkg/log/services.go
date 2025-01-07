package log

import (
	"fmt"

	"github.com/samber/do"
	"github.com/sirupsen/logrus"
	"github.com/zhulik/fid/pkg/core"
)

func Register(injector *do.Injector) {
	do.Provide[logrus.FieldLogger](injector, func(_ *do.Injector) (logrus.FieldLogger, error) {
		config, err := do.Invoke[core.Config](injector)
		if err != nil {
			return nil, err
		}

		logger := logrus.New()

		logLevel, err := logrus.ParseLevel(config.LogLevel())
		if err != nil {
			return nil, fmt.Errorf("failed parse loglevel: %w", err)
		}

		logger.SetLevel(logLevel)

		return logger, nil
	})
}
