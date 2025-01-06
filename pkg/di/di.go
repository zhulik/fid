package di

import (
	"github.com/samber/do"
	"github.com/zhulik/fid/pkg/config"
)

func New() *do.Injector {
	injector := do.New()

	config.Register(injector)

	// TODO: inject config

	return injector
}
