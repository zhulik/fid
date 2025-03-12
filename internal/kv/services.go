package kv

import (
	"context"

	"github.com/samber/do"
	"github.com/zhulik/fid/internal/core"
	"github.com/zhulik/fid/internal/kv/nats"
)

func Register(ctx context.Context) {
	do.Provide(nil, func(injector *do.Injector) (core.KV, error) {
		return nats.NewKV(injector)
	})
}
