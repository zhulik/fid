package proxyserver

import (
	"github.com/samber/do"
)

func Register(injector *do.Injector) {
	do.Provide(injector, NewServer)
}
