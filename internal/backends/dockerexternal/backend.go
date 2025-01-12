package dockerexternal

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/samber/do"
	"github.com/sirupsen/logrus"
	"github.com/zhulik/fid/internal/core"
	"github.com/zhulik/fid/pkg/iter"
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

	logger = logger.WithField("component", "backends.dockerexternal.Backend")

	defer logger.Info("ContainerBackend created.")

	// TODO: validate function configs

	backend := Backend{
		docker: docker,

		logger: logger,
	}

	_, err = backend.Functions(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to fetch functions: %w", err)
	}

	return &backend, nil
}

func (b Backend) Info(ctx context.Context) (map[string]any, error) {
	info, err := b.docker.Info(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to docker info: %w", err)
	}

	return map[string]any{
		"backend":      "Docker external backend",
		"dockerEngine": info,
	}, nil
}

func (b Backend) Function(ctx context.Context, name string) (core.Function, error) { //nolint:ireturn
	b.logger.WithField("function", name).Debug("Fetching function info")

	fnFilters := filters.NewArgs()
	fnFilters.Add("label", fmt.Sprintf("%s=%s", core.LabelNameComponent, core.FunctionComponentLabelValue))
	fnFilters.Add("name", name)

	containers, err := b.docker.ContainerList(ctx, container.ListOptions{Filters: fnFilters})
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	if len(containers) == 0 {
		return nil, core.ErrFunctionNotFound
	}

	inspect, err := b.docker.ContainerInspect(ctx, containers[0].ID)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect container: %w", err)
	}

	function, err := NewFunction(inspect)
	if err != nil {
		return nil, fmt.Errorf("failed to create function from container: %w", err)
	}

	return function, nil
}

func (b Backend) Functions(ctx context.Context) ([]core.Function, error) {
	b.logger.Debug("Fetching function list")

	fnFilters := filters.NewArgs()
	fnFilters.Add("label", fmt.Sprintf("%s=%s", core.LabelNameComponent, core.FunctionComponentLabelValue))

	containers, err := b.docker.ContainerList(ctx, container.ListOptions{Filters: fnFilters})
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	// TODO: map in parallel
	functions, err := iter.MapErr(containers, func(t types.Container) (core.Function, error) {
		inspect, err := b.docker.ContainerInspect(ctx, t.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to inspect container: %w", err)
		}

		return NewFunction(inspect)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to map containers to functions: %w", err)
	}

	return functions, nil
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
