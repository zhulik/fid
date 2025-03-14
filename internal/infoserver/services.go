package infoserver

import (
	"context"

	"github.com/samber/do"
)

func Register(ctx context.Context, injector *do.Injector) {
	do.Provide(injector, NewServer)
}
