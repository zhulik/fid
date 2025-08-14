package di

import (
	"context"
	"fmt"

	"github.com/zhulik/fid/internal/backends"
	"github.com/zhulik/fid/internal/config"
	"github.com/zhulik/fid/internal/gateway"
	"github.com/zhulik/fid/internal/infoserver"
	"github.com/zhulik/fid/internal/invocation"
	"github.com/zhulik/fid/internal/kv"
	"github.com/zhulik/fid/internal/logging"
	"github.com/zhulik/fid/internal/pubsub"
	"github.com/zhulik/fid/internal/runtimeapi"
	"github.com/zhulik/fid/internal/scaler"
	"github.com/zhulik/pal"
)

func InitPal(ctx context.Context, cfg *config.Config) (*pal.Pal, error) {
	p := pal.New(
		pal.Provide(cfg),
		logging.Provide(),
		runtimeapi.Provide(),
		backends.Provide(),
		gateway.Provide(),
		pubsub.Provide(),
		kv.Provide(),
		invocation.Provide(),
		infoserver.Provide(),
		scaler.Provide(),
	)

	err := p.Init(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize pal: %w", err)
	}

	return p, nil
}
