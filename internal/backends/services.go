package backends

import (
	"github.com/docker/docker/client"
	"github.com/samber/do"
	"github.com/zhulik/fid/internal/backends/docker"
	"github.com/zhulik/fid/internal/core"
)

func Register() {
	do.Provide(nil, func(injector *do.Injector) (*client.Client, error) {
		return client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	})

	do.Provide(nil, func(injector *do.Injector) (core.ContainerBackend, error) {
		return docker.New(injector)
	})
}
