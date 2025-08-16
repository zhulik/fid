package invocation

import (
	"github.com/zhulik/fid/internal/core"
	"github.com/zhulik/pal"
)

func Provide() pal.ServiceDef {
	return pal.ProvideList(
		pal.Provide[core.Invoker](&Invoker{}),
	)
}
