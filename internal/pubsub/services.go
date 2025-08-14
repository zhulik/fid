package pubsub

import (
	"context"

	"github.com/samber/do/v2"
	"github.com/zhulik/fid/internal/core"
	"github.com/zhulik/fid/internal/pubsub/nats"
	"github.com/zhulik/pal"
)

func Provide() pal.ServiceDef {
	return pal.ProvideList(
		pal.Provide(&nats.Client{}),
		pal.Provide[core.PubSuber](&nats.PubSuber{}),
	)
}

func Register(ctx context.Context, injector do.Injector) {
	do.Provide(injector, nats.NewClient)
	do.Provide(injector, func(injector do.Injector) (core.PubSuber, error) {
		return nats.NewPubSuber(injector)
	})
}
