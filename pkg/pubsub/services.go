package pubsub

import (
	"github.com/samber/do"
	"github.com/zhulik/fid/pkg/core"
	"github.com/zhulik/fid/pkg/pubsub/nats"
)

func Register(injector *do.Injector) {
	do.Provide(injector, nats.NewClient)

	do.Provide(injector, func(injector *do.Injector) (core.Publisher, error) {
		return nats.NewPublisher(injector)
	})
}
