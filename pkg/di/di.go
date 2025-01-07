package di

import (
	"github.com/samber/do"
	"github.com/zhulik/fid/pkg/backends"
	"github.com/zhulik/fid/pkg/config"
	"github.com/zhulik/fid/pkg/infoserver"
	"github.com/zhulik/fid/pkg/log"
	"github.com/zhulik/fid/pkg/proxyserver"
	"github.com/zhulik/fid/pkg/pubsub"
	"github.com/zhulik/fid/pkg/wsserver"
)

func New() *do.Injector {
	injector := do.New()

	config.Register(injector)
	log.Register(injector)

	wsserver.Register(injector)
	backends.Register(injector)
	proxyserver.Register(injector)
	pubsub.Register(injector)
	infoserver.Register(injector)

	// TODO: inject config

	return injector
}
