package pubsub

import (
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
