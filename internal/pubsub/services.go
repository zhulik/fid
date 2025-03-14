package pubsub

import (
	"context"

	"github.com/samber/do"
	"github.com/zhulik/fid/internal/core"
	"github.com/zhulik/fid/internal/pubsub/nats"
)

func Register(ctx context.Context, injector *do.Injector) {
	do.Provide(injector, nats.NewClient)
	do.Provide(injector, func(injector *do.Injector) (core.PubSuber, error) {
		return nats.NewPubSuber(injector)
	})
}
