package wsserver

import (
	"github.com/samber/do"
)

func Register(injector *do.Injector) {
	do.Provide(injector, func(injector *do.Injector) (*Server, error) {
		return NewServer(injector)
	})
}
