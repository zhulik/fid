package invocation

import (
	"github.com/samber/do"
	"github.com/zhulik/fid/internal/core"
)

func Register(injector *do.Injector) {
	do.Provide(injector, func(injector *do.Injector) (core.Invoker, error) {
		return NewInvoker(injector)
	})
}
