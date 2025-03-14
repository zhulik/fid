package runtimeapi

import (
	"context"

	"github.com/samber/do"
)

func Register(ctx context.Context, injector *do.Injector) {
	do.Provide(injector, func(injector *do.Injector) (*Server, error) {
		return NewServer(ctx, injector)
	})
}
