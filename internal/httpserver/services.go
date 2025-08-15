package httpserver

import (
	"github.com/zhulik/pal"
)

func Provide() pal.ServiceDef {
	return pal.Provide(&Server{})
}
