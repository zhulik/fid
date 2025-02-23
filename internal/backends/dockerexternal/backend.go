package dockerexternal

import (
	"context"
	"fmt"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
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

// Register creates a new function template container, scaler, forwarder(TODO) and garbage collector(TODO).
func (b Backend) Register(ctx context.Context, function core.Function) error {
	err := b.createFunctionTemplate(ctx, function)
	if err != nil {
		return err
	}

	err = b.createScaler(ctx, function)
	if err != nil {
		return err
	}

	return nil
}

func (b Backend) createScaler(ctx context.Context, function core.Function) error {
	logger := b.logger.WithField("function", function.Name())

	containerConfig := &container.Config{
		Image: core.ImageNameRuntimeAPI,
		Env: []string{
			fmt.Sprintf("%s=%s", core.EnvNameFunctionName, function.Name()),
			// TODO: get this value from somewhere else, remove hardcoded value
			fmt.Sprintf("%s=%s", core.EnvNameNatsURL, "nats://nats:4222"),
		},
		Labels: map[string]string{
			core.LabelNameComponent: core.ScalerComponentLabelValue,
		},
	}

	hostConfig := &container.HostConfig{
		Binds: []string{
			"/var/run/docker.sock:/var/run/docker.sock", // TODO: configurable
		},
		AutoRemove: true,
	}
	networkingConfig := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			"nats": {}, // TODO: get from config
		},
	}

	containerName := function.Name() + "-scaler"

	_, err := b.docker.ContainerCreate(ctx, containerConfig, hostConfig, networkingConfig, nil, containerName)
	if err != nil {
		if strings.Contains(err.Error(), "Conflict. The container name") {
			logger.Infof("Scaler container already exists")
			return nil
		}
		return fmt.Errorf("failed to create scaler container: %w", err)
	}

	err = b.docker.ContainerStart(ctx, containerName, container.StartOptions{})
	if err != nil {
		return fmt.Errorf("failed to start scaler container: %w", err)
	}

	logger.Infof("Scaler container created and started")

	return nil
}

func (b Backend) createFunctionTemplate(ctx context.Context, function core.Function) error {
	// Using a stopped container as a template is not really a good idea, but we can change it to KV later.
	err := b.docker.ContainerRemove(ctx, function.Name(), container.RemoveOptions{
		Force: true,
	})

	logger := b.logger.WithField("function", function.Name())

	if err != nil {
		if client.IsErrNotFound(err) {
			logger.Infof("Creating function template container")
		} else {
			return fmt.Errorf("failed to remove function template container for '%s': %w", function.Name(), err)
		}
	} else {
		logger.Info("Recreating function template container")
	}

	vars := []string{}
	for k, v := range function.Env() {
		vars = append(vars, fmt.Sprintf("%s=%s", k, v))
	}

	containerConfig := &container.Config{
		Image: core.ImageNameRuntimeAPI,
		Env:   vars,
		Labels: map[string]string{
			core.LabelNameComponent: core.FunctionTemplateComponentLabelValue,
		},
	}
	hostConfig := &container.HostConfig{}
	networkingConfig := &network.NetworkingConfig{}

	_, err = b.docker.ContainerCreate(ctx, containerConfig, hostConfig, networkingConfig, nil, function.Name())
	if err != nil {
		return fmt.Errorf("failed to create function template container: %w", err)
	}

	logger.Infof("Function template container created")

	return nil
}

func New(docker *client.Client, injector *do.Injector) (*Backend, error) {
	logger, err := do.Invoke[logrus.FieldLogger](injector)
	if err != nil {
		return nil, err
	}

	logger = logger.WithField("component", "backends.dockerexternal.Backend")

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
		"backend":      "Docker external backend",
		"dockerEngine": info,
	}, nil
}

func (b Backend) Function(ctx context.Context, name string) (core.Function, error) {
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
	b.logger.Infof("Creating new function pod for function %s", function.Name())

	pod, err := CreateFunctionPod(ctx, b.docker, function)
	if err != nil {
		return "", err
	}

	b.logger.Infof("Function pod function %s created id=%s", function.Name(), pod.UUID)

	return pod.UUID, nil
}

func (b Backend) KillInstance(ctx context.Context, instanceID string) error {
	b.logger.Infof("Killing function instance %s", instanceID)

	return FunctionPod{UUID: instanceID, docker: b.docker}.Delete(ctx)
}
