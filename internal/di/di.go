package di

import (
	"github.com/samber/do"
	"github.com/zhulik/fid/internal/backends"
	"github.com/zhulik/fid/internal/config"
	"github.com/zhulik/fid/internal/forwarder"
	"github.com/zhulik/fid/internal/gateway"
	"github.com/zhulik/fid/internal/infoserver"
	"github.com/zhulik/fid/internal/log"
	"github.com/zhulik/fid/internal/pubsub"
)

func New() *do.Injector {
	injector := do.New()

	config.Register(injector)
	log.Register(injector)

	forwarder.Register(injector)
	backends.Register(injector)
	gateway.Register(injector)
	pubsub.Register(injector)
	infoserver.Register(injector)

	// TODO: inject config

	return injector
}
