package backends

import (
	"context"

	"github.com/docker/docker/client"
	"github.com/samber/do"
	"github.com/zhulik/fid/internal/backends/docker"
	"github.com/zhulik/fid/internal/core"
)

func Register(ctx context.Context) {
	do.Provide(nil, func(injector *do.Injector) (*client.Client, error) {
		return client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	})

	do.Provide(nil, func(injector *do.Injector) (core.ContainerBackend, error) {
		return docker.New(injector)
	})

	do.Provide(nil, func(injector *do.Injector) (core.FunctionsRepo, error) {
		return docker.NewFunctionsRepo(ctx, injector)
	})

	do.Provide(nil, func(injector *do.Injector) (core.InstancesRepo, error) {
		return docker.NewInstancesRepo(ctx, injector)
	})
}
