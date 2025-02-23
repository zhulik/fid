package kv

import (
	"github.com/samber/do"
	"github.com/zhulik/fid/internal/core"
	"github.com/zhulik/fid/internal/kv/nats"
)

func Register() {
	do.Provide(nil, func(injector *do.Injector) (core.KV, error) {
		return nats.NewKV(injector)
	})
}
