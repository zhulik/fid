package backends

import (
	"context"
	"errors"
	"github.com/zhulik/fid/pkg/core"
	"os"

	"github.com/docker/docker/client"
	"github.com/docker/docker/errdefs"

	"github.com/samber/do"

	"github.com/zhulik/fid/pkg/backends/dockerexternal"
)

var (
	ErrCannotDetectBackend = errors.New("cannot detect backend")
)

func Register(injector *do.Injector) {
	// Currently it tries to detect your backend. In the future it should use external config.
	do.Provide(injector, func(injector *do.Injector) (core.Backend, error) {
		hostname, err := os.Hostname()
		if err != nil {
			return nil, err
		}

		cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
		if err != nil {
			return nil, err
		}

		_, err = cli.ContainerInspect(context.Background(), hostname)
		if err != nil {
			if errdefs.IsNotFound(err) {
				// We can connect to the Docker daemon, but current machine's hostname is not a container ID.
				// Using external docker backend

				return dockerexternal.New(cli), nil
			}

			if client.IsErrConnectionFailed(err) {
				// We cannot connect to the Docker daemon.
				return nil, ErrCannotDetectBackend
			}

			return nil, err
		}

		return nil, ErrCannotDetectBackend // TODO: internal docker backend
	})
}
