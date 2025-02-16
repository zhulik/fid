package dockerexternal

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/google/uuid"
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

	function, err := NewFunction(inspect)
	if err != nil {
		return nil, fmt.Errorf("failed to create function from container: %w", err)
	}

	return function, nil
}

func (b Backend) Functions(ctx context.Context) ([]core.Function, error) {
	b.logger.Debug("Fetching function list")

	fnFilters := filters.NewArgs()
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

func (b Backend) AddInstance(ctx context.Context, function core.Function) (string, error) {
	instanceID := uuid.NewString()

	networkID, err := b.createNetwork(ctx, function, instanceID)
	if err != nil {
		return "", err
	}

	err = b.createForwarder(ctx, function, instanceID, networkID)
	if err != nil {
		return "", err
	}

	return instanceID, nil
}

func (b Backend) createForwarder(ctx context.Context, function core.Function, instanceID string, networkID string) error { //nolint:lll
	containerName := b.forwarderContainerName(function, instanceID)

	containerConfig := &container.Config{
		Image: core.ImageNameForwarder,
		Env: []string{
			fmt.Sprintf("%s=%s", core.EnvNameFunctionName, function.Name()),
			fmt.Sprintf("%s=%s", core.EnvNameInstanceID, instanceID),
			"NATS_URL=" + "nats://nats:4222", // TODO: get this value from somewhere else, remove hardcoded value
		},
		Labels: map[string]string{
			core.LabelNameComponent: core.ForwarderComponentLabelValue,
		},
	}
	hostConfig := &container.HostConfig{
		Binds: []string{
			"/var/run/docker.sock:/var/run/docker.sock", // Replace with your paths
		},
		AutoRemove: true,
	}
	networkingConfig := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			networkID: {},
			"nats":    {}, // TODO: get this network name from somewhere else, remove hardcoded value
		},
	}

	b.logger.Debugf("Creating forwarder container '%s'.", containerName)

	resp, err := b.docker.ContainerCreate(ctx, containerConfig, hostConfig, networkingConfig, nil, containerName)
	if err != nil {
		return fmt.Errorf("failed to create container: %w", err)
	}

	b.logger.Debugf("Forwarder container created '%s'.", containerName)

	err = b.docker.ContainerStart(ctx, resp.ID, container.StartOptions{})
	if err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}

	b.logger.Infof("Forwarder container started '%s'.", containerName)

	return nil
}

func (b Backend) forwarderContainerName(function core.Function, instanceID string) string {
	return fmt.Sprintf("fid-%s-%s-forwarder", function.Name(), instanceID)
}

func (b Backend) createNetwork(ctx context.Context, function core.Function, instanceID string) (string, error) {
	networkName := networkName(function, instanceID)

	b.logger.Debug("Creating network '%s' for function '%s' instance '%s'.", networkName, function.Name(), instanceID)

	networkResp, err := b.docker.NetworkCreate(ctx, networkName, network.CreateOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to create network '%s': %w", networkName, err)
	}

	b.logger.Infof("Network created '%s', id=%s.", networkName, networkResp.ID)

	return networkResp.ID, nil
}

func (b Backend) KillInstance(ctx context.Context, function core.Function, instanceID string) error {
	containerName := b.forwarderContainerName(function, instanceID)

	err := b.docker.ContainerStop(ctx, containerName, container.StopOptions{})
	if err != nil {
		return err
	}

	b.logger.Infof("Forwarder container stopped '%s'.", containerName)

	err = b.deleteNetwork(ctx, function, instanceID)
	if err != nil {
		return err
	}

	return nil
}

func (b Backend) deleteNetwork(ctx context.Context, function core.Function, instanceID string) error {
	networks, err := b.docker.NetworkList(ctx, network.ListOptions{
		Filters: filters.NewArgs(filters.Arg("name", networkName(function, instanceID))),
	})
	if err != nil {
		return fmt.Errorf("failed to list networks: %w", err)
	}

	if len(networks) == 0 {
		return core.ErrInstanceNotFound
	}

	b.logger.Debugf("Removing network '%s'.", networks[0].Name)

	err = b.docker.NetworkRemove(ctx, networks[0].ID)
	if err != nil {
		return fmt.Errorf("failed to remove network '%s': %w", networks[0].Name, err)
	}

	b.logger.Infof("Network deleted '%s', id=%s.", networks[0].Name, networks[0].ID)

	return nil
}

func networkName(function core.Function, instanceID string) string {
	return fmt.Sprintf("fid-%s-%s", function.Name(), instanceID)
}
