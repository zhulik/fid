package pubsub

import (
	"github.com/samber/do"
	"github.com/zhulik/fid/internal/core"
	nats2 "github.com/zhulik/fid/internal/pubsub/nats"
)

func Register(injector *do.Injector) {
	do.Provide(injector, nats2.NewClient)

	do.Provide(injector, func(injector *do.Injector) (core.Publisher, error) {
		return nats2.NewPublisher(injector)
	})

	do.Provide(injector, func(injector *do.Injector) (core.Subscriber, error) {
		return nats2.NewSubscriber(injector)
	})
}
