package config

import (
	"github.com/caarlos0/env/v11"
	"github.com/samber/do"
	"github.com/zhulik/fid/pkg/core"
)

func Register(injector *do.Injector) {
	// Currently it tries to detect your backend. In the future it should use external config.
	do.Provide(injector, func(_ *do.Injector) (core.Config, error) {
		var cfg Config

		if err := env.Parse(&cfg); err != nil {
			return nil, err
		}

		return cfg, nil
	})
}
