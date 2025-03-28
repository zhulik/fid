package backends

import (
	"context"

	"github.com/docker/docker/client"
	"github.com/samber/do/v2"
	"github.com/zhulik/fid/internal/backends/docker"
	"github.com/zhulik/fid/internal/core"
)

func Register(ctx context.Context, injector do.Injector) {
	do.Provide(injector, func(injector do.Injector) (*client.Client, error) {
		return client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	})

	do.Provide(injector, func(injector do.Injector) (core.ContainerBackend, error) {
		return docker.New(injector)
	})

	do.Provide(injector, func(injector do.Injector) (core.FunctionsRepo, error) {
		return docker.NewFunctionsRepo(ctx, injector)
	})

	do.Provide(injector, func(injector do.Injector) (core.InstancesRepo, error) {
		return docker.NewInstancesRepo(ctx, injector)
	})
}
