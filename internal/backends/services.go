package backends

import (
	"fmt"

	"github.com/docker/docker/client"
	"github.com/samber/do"
	"github.com/zhulik/fid/internal/backends/docker"
	"github.com/zhulik/fid/internal/core"
)

func Register() {
	// Currently it tries to detect your backend. In the future it should use external config.
	do.Provide(nil, func(injector *do.Injector) (core.ContainerBackend, error) {
		cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
		if err != nil {
			return nil, fmt.Errorf("failed to build docker client: %w", err)
		}

		return docker.New(cli, injector)
	})
}
