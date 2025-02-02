package kv

import (
	"github.com/samber/do"
	"github.com/zhulik/fid/internal/core"
	"github.com/zhulik/fid/internal/kv/nats"
)

func Register(injector *do.Injector) {
	do.Provide(injector, func(injector *do.Injector) (core.KV, error) {
		return nats.NewKV(injector)
	})
}
