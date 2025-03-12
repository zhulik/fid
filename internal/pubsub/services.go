package pubsub

import (
	"context"

	"github.com/samber/do"
	"github.com/zhulik/fid/internal/core"
	"github.com/zhulik/fid/internal/pubsub/nats"
)

func Register(ctx context.Context) {
	do.Provide(nil, nats.NewClient)
	do.Provide[core.PubSuber](nil, func(injector *do.Injector) (core.PubSuber, error) {
		return nats.NewPubSuber(injector)
	})
}
