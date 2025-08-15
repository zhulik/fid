package backends

import (
	"context"

	"github.com/docker/docker/client"
	"github.com/zhulik/fid/internal/backends/docker"
	"github.com/zhulik/fid/internal/core"
	"github.com/zhulik/pal"
)

func Provide() pal.ServiceDef {
	return pal.ProvideList(
		pal.ProvideFn(func(ctx context.Context) (*client.Client, error) {
			return client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
		}),
		pal.Provide[core.ContainerBackend](&docker.Backend{}),
		pal.Provide[core.FunctionsRepo](&docker.FunctionsRepo{}),
		pal.Provide[core.InstancesRepo](&docker.InstancesRepo{}),
	)
}
