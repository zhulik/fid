package backends

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/docker/docker/client"
	"github.com/docker/docker/errdefs"
	"github.com/samber/do"
	"github.com/zhulik/fid/internal/backends/dockerexternal"
	"github.com/zhulik/fid/internal/backends/dockerinternal"
	"github.com/zhulik/fid/internal/core"
)

var ErrCannotDetectBackend = errors.New("cannot detect backend")

func Register(injector *do.Injector) {
	// Currently it tries to detect your backend. In the future it should use external config.
	do.Provide(injector, func(_ *do.Injector) (core.ContainerBackend, error) {
		hostname, err := os.Hostname()
		if err != nil {
			return nil, fmt.Errorf("failed to get hostname: %w", err)
		}

		cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
		if err != nil {
			return nil, fmt.Errorf("failed to build docker client: %w", err)
		}

		_, err = cli.ContainerInspect(context.Background(), hostname)
		if err != nil {
			if errdefs.IsNotFound(err) {
				// We can connect to the Docker daemon, but current machine's hostname is not a container ID.
				// Using external docker backend
				return dockerexternal.New(cli, injector)
			}

			if client.IsErrConnectionFailed(err) {
				// We cannot connect to the Docker daemon.
				return nil, ErrCannotDetectBackend
			}

			return nil, fmt.Errorf("failed inspect docker container: %w", err)
		}

		return dockerinternal.New(cli, injector)
	})
}
