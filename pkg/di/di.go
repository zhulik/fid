package di

import (
	"github.com/samber/do"
	"github.com/zhulik/fid/pkg/config"
	"github.com/zhulik/fid/pkg/log"
)

func New() *do.Injector {
	injector := do.New()

	log.Register(injector)
	config.Register(injector)

	// TODO: inject config

	return injector
}
