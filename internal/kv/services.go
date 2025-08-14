package kv

import (
	"context"

	"github.com/samber/do/v2"
	"github.com/zhulik/fid/internal/core"
	"github.com/zhulik/fid/internal/kv/nats"
	"github.com/zhulik/pal"
)

func Provide() pal.ServiceDef {
	return pal.ProvideList(
		pal.Provide[core.KV](&nats.KV{}),
	)
}

func Register(ctx context.Context, injector do.Injector) {
	do.Provide(injector, func(injector do.Injector) (core.KV, error) {
		return nats.NewKV(injector)
	})
}
