package gateway

import (
	"context"

	"github.com/samber/do/v2"
)

func Register(ctx context.Context, injector do.Injector) {
	do.Provide(injector, NewServer)
}
