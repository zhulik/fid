package dockerexternal

import (
	"context"
	"errors"
	"github.com/docker/docker/client"
	"github.com/zhulik/fid/pkg/core"

	"github.com/zhulik/fid/pkg/log"
)

var (
	logger = log.Logger.WithField("component", "backends.dockerexternal.Backend")

	ErrFunctionNotFound = errors.New("function not found")
)

type Backend struct {
	docker *client.Client
}

func New(docker *client.Client) core.Backend {
	logger.Info("Creating new backend...")
	defer logger.Info("Backend created.")

	return Backend{
		docker: docker,
	}
}

func (b Backend) Info(ctx context.Context) (map[string]any, error) {
	info, err := b.docker.Info(context.Background())

	if err != nil {
		return nil, err
	}

	return map[string]any{
		"backend":      "Docker external backend",
		"dockerEngine": info,
	}, nil
}

func (b Backend) Function(ctx context.Context, name string) (core.Function, error) {
	return nil, ErrFunctionNotFound
}

func (b Backend) Functions(ctx context.Context) ([]core.Function, error) {
	return []core.Function{}, nil
}

func (b Backend) HealthCheck() error {
	logger.Info("Backend health check.")
	_, err := b.docker.Info(context.Background())
	return err
}

func (b Backend) Shutdown() error {
	logger.Info("Backend shutting down...")
	defer logger.Info("Backend shot down.")

	return b.docker.Close()
}
