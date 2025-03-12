package invocation

import (
	"context"

	"github.com/samber/do"
	"github.com/zhulik/fid/internal/core"
)

func Register(ctx context.Context) {
	do.Provide(nil, func(injector *do.Injector) (core.Invoker, error) {
		return NewInvoker(injector)
	})
}
