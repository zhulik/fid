package testhelpers

import (
	"context"
	"time"

	"github.com/samber/lo"
	"github.com/zhulik/fid/internal/config"
	"github.com/zhulik/fid/internal/logging"
	pubSubNats "github.com/zhulik/fid/internal/pubsub/nats"
	"github.com/zhulik/pal"
)

//nolint:mnd
func NewPal(ctx context.Context, services ...pal.ServiceDef) *pal.Pal {
	services = append(services,
		logging.Provide(),
		pal.Provide(&config.Config{}),
		pal.Provide(&pubSubNats.Client{}),
	)

	p := pal.New(
		services...,
	).
		InitTimeout(time.Second * 10).
		HealthCheckTimeout(time.Second * 10).
		ShutdownTimeout(time.Second * 10)

	lo.Must0(p.Init(ctx))

	return p
}
