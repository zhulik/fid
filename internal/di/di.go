package di

import (
	"context"

	"github.com/samber/do/v2"
	"github.com/zhulik/fid/internal/backends"
	"github.com/zhulik/fid/internal/gateway"
	"github.com/zhulik/fid/internal/infoserver"
	"github.com/zhulik/fid/internal/invocation"
	"github.com/zhulik/fid/internal/kv"
	"github.com/zhulik/fid/internal/logging"
	"github.com/zhulik/fid/internal/pubsub"
	"github.com/zhulik/fid/internal/runtimeapi"
	"github.com/zhulik/fid/internal/scaler"
)

func Init() *do.RootScope {
	injector := do.New()

	ctx := context.Background()

	logging.Register(ctx, injector)
	runtimeapi.Register(ctx, injector)
	backends.Register(ctx, injector)
	gateway.Register(ctx, injector)
	pubsub.Register(ctx, injector)
	kv.Register(ctx, injector)
	invocation.Register(ctx, injector)
	infoserver.Register(ctx, injector)
	scaler.Register(ctx, injector)

	return injector
}
