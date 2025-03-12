package scaler

import (
	"context"

	"github.com/samber/do"
)

func Register(ctx context.Context) {
	do.Provide(nil, func(injector *do.Injector) (*Server, error) {
		return NewServer(ctx, injector)
	})
}
