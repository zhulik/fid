package runtimeapi

import (
	"context"

	"github.com/samber/do/v2"
	"github.com/zhulik/pal"
)

func Provide() pal.ServiceDef {
	return pal.ProvideList(
		pal.Provide(&Server{}),
	)
}

func Register(ctx context.Context, injector do.Injector) {
	do.Provide(injector, func(injector do.Injector) (*Server, error) {
		return NewServer(ctx, injector)
	})
}
