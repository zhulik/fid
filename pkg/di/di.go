package di

import (
	"github.com/samber/do"
	"github.com/zhulik/fid/pkg/backends"
	"github.com/zhulik/fid/pkg/config"
	"github.com/zhulik/fid/pkg/forwarder"
	"github.com/zhulik/fid/pkg/gateway"
	"github.com/zhulik/fid/pkg/infoserver"
	"github.com/zhulik/fid/pkg/log"
	"github.com/zhulik/fid/pkg/pubsub"
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
