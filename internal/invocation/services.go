package invocation

import (
	"context"

	"github.com/samber/do/v2"
	"github.com/zhulik/fid/internal/core"
	"github.com/zhulik/pal"
)

func Provide() pal.ServiceDef {
	return pal.ProvideList(
		pal.Provide[core.Invoker](&Invoker{}),
	)
}

func Register(ctx context.Context, injector do.Injector) {
	do.Provide(injector, func(injector do.Injector) (core.Invoker, error) {
		return NewInvoker(injector)
	})
}
