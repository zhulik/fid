package dockerinternal

import (
	"context"

	"github.com/docker/docker/client"
	"github.com/samber/do"
	"github.com/sirupsen/logrus"
	"github.com/zhulik/fid/pkg/core"
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

	logger.Info("Creating new backend...")
	defer logger.Info("ContainerBackend created.")

	return &Backend{
		docker: docker,
		logger: logger,
	}, nil
}

func (b Backend) Info(ctx context.Context) (map[string]any, error) {
	info, err := b.docker.Info(ctx)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"backend":      "Docker internal backend",
		"dockerEngine": info,
	}, nil
}

func (b Backend) Function(_ context.Context, _ string) (core.Function, error) { //nolint:ireturn
	return nil, core.ErrFunctionNotFound
}

func (b Backend) Functions(_ context.Context) ([]core.Function, error) {
	return []core.Function{}, nil
}

func (b Backend) HealthCheck() error {
	b.logger.Debug("ContainerBackend health check.")

	_, err := b.docker.Info(context.Background())

	return err
}

func (b Backend) Shutdown() error {
	b.logger.Info("ContainerBackend shutting down...")
	defer b.logger.Info("ContainerBackend shot down.")

	return b.docker.Close()
}
