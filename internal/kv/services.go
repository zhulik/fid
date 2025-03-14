package kv

import (
	"context"

	"github.com/samber/do/v2"
	"github.com/zhulik/fid/internal/core"
	"github.com/zhulik/fid/internal/kv/nats"
)

func Register(ctx context.Context, injector do.Injector) {
	do.Provide(injector, func(injector do.Injector) (core.KV, error) {
		return nats.NewKV(injector)
	})
}
