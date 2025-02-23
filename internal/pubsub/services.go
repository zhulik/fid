package pubsub

import (
	"github.com/samber/do"
	"github.com/zhulik/fid/internal/core"
	"github.com/zhulik/fid/internal/pubsub/nats"
)

func Register() {
	do.Provide(nil, nats.NewClient)
	do.Provide[core.PubSuber](nil, func(injector *do.Injector) (core.PubSuber, error) {
		return nats.NewPubSuber(injector)
	})
}
