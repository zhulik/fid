package config

import (
	"fmt"

	"github.com/caarlos0/env/v11"
	"github.com/samber/do"
	"github.com/zhulik/fid/internal/core"
)

func Register() {
	do.Provide(nil, func(_ *do.Injector) (core.Config, error) {
		var cfg Config

		if err := env.Parse(&cfg); err != nil {
			return nil, fmt.Errorf("failed to parse config from ENV: %w", err)
		}

		return cfg, nil
	})
}
