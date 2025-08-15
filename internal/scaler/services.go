package scaler

import (
	"github.com/zhulik/pal"
)

func Provide() pal.ServiceDef {
	return pal.ProvideList(
		pal.Provide(&Server{}),
		pal.Provide(&Scaler{}),
	)
}
