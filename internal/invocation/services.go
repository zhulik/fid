package invocation

import (
	"context"

	"github.com/samber/do"
	"github.com/zhulik/fid/internal/core"
)

func Register(ctx context.Context, injector *do.Injector) {
	do.Provide(injector, func(injector *do.Injector) (core.Invoker, error) {
		return NewInvoker(injector)
	})
}
