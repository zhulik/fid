package gateway

import (
	"github.com/zhulik/pal"
)

func Provide() pal.ServiceDef {
	return pal.ProvideList(
		pal.Provide(&Server{}),
	)
}
