package dockerinternal

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/samber/do"
	"github.com/sirupsen/logrus"
	"github.com/zhulik/fid/internal/backends/dockerexternal"
	"github.com/zhulik/fid/internal/core"
	"github.com/zhulik/fid/pkg/iter"
)

type Backend struct {
	docker *client.Client

	logger logrus.FieldLogger
}

func (b Backend) AddInstance(ctx context.Context, function core.Function) (string, error) {
	panic("implement me")
}

func (b Backend) KillInstance(ctx context.Context, function core.Function, instanceID string) error {
	panic("implement me")
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

func (b Backend) Function(ctx context.Context, name string) (core.Function, error) {
	b.logger.WithField("function", name).Debug("Fetching function info")

	fnFilters := filters.NewArgs()
	// TODO: filter by compose project.
	fnFilters.Add("label", fmt.Sprintf("%s=%s", core.LabelNameComponent, core.FunctionTemplateComponentLabelValue))
	fnFilters.Add("label", fmt.Sprintf("%s=%s", core.LabelNameFunctionName, name))

	containers, err := b.docker.ContainerList(ctx, container.ListOptions{Filters: fnFilters, All: true})
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

	function, err := dockerexternal.NewFunction(inspect)
	if err != nil {
		return nil, fmt.Errorf("failed to create function from container: %w", err)
	}

	return function, nil
}

func (b Backend) Functions(ctx context.Context) ([]core.Function, error) {
	b.logger.Debug("Fetching function list")

	fnFilters := filters.NewArgs()
	// TODO: filter by compose project.
	fnFilters.Add("label", fmt.Sprintf("%s=%s", core.LabelNameComponent, core.FunctionTemplateComponentLabelValue))

	containers, err := b.docker.ContainerList(ctx, container.ListOptions{Filters: fnFilters, All: true})
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	// TODO: map in parallel
	functions, err := iter.MapErr(containers, func(t types.Container) (core.Function, error) {
		inspect, err := b.docker.ContainerInspect(ctx, t.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to inspect container: %w", err)
		}

		return dockerexternal.NewFunction(inspect)
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
