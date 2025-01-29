package pubsub

import (
	"github.com/samber/do"
	"github.com/zhulik/fid/internal/core"
	"github.com/zhulik/fid/internal/pubsub/nats"
)

func Register(injector *do.Injector) {
	do.Provide(injector, nats.NewClient)
	do.Provide(injector, func(injector *do.Injector) (core.Publisher, error) {
		return nats.NewPublisher(injector)
	})
	do.Provide(injector, func(injector *do.Injector) (core.Subscriber, error) {
		return nats.NewSubscriber(injector)
	})
}
