package dockerinternal

import (
	"context"
	"fmt"

	"github.com/docker/docker/client"
	"github.com/samber/do"
	"github.com/sirupsen/logrus"
	core2 "github.com/zhulik/fid/internal/core"
)

type Backend struct {
	docker *client.Client

	logger logrus.FieldLogger
}

func New(docker *client.Client, injector *do.Injector) (*Backend, error) {
	logger, err := do.Invoke[logrus.FieldLogger](injector)
	if err != nil {
		return nil, err
	}

	logger = logger.WithField("component", "backends.dockerinternal.Backend")

	defer logger.Info("ContainerBackend created.")

	return &Backend{
		docker: docker,
		logger: logger,
	}, nil
}

func (b Backend) Info(ctx context.Context) (map[string]any, error) {
	info, err := b.docker.Info(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to docker info: %w", err)
	}

	return map[string]any{
		"backend":      "Docker internal backend",
		"dockerEngine": info,
	}, nil
}

func (b Backend) Function(_ context.Context, _ string) (core2.Function, error) { //nolint:ireturn
	return nil, core2.ErrFunctionNotFound
}

func (b Backend) Functions(_ context.Context) ([]core2.Function, error) {
	return []core2.Function{}, nil
}

func (b Backend) HealthCheck() error {
	b.logger.Debug("ContainerBackend health check.")

	_, err := b.docker.Info(context.Background())
	if err != nil {
		return fmt.Errorf("backend health check failed: %w", err)
	}

	return nil
}

func (b Backend) Shutdown() error {
	b.logger.Debug("ContainerBackend shutting down...")
	defer b.logger.Debug("ContainerBackend shot down.")

	err := b.docker.Close()
	if err != nil {
		return fmt.Errorf("failed to shut down the backend: %w", err)
	}

	return nil
}
