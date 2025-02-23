package invocation

import (
	"github.com/samber/do"
	"github.com/zhulik/fid/internal/core"
)

func Register() {
	do.Provide(nil, func(injector *do.Injector) (core.Invoker, error) {
		return NewInvoker(injector)
	})
}
