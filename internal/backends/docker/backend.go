package docker

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
	config core.Config
	logger logrus.FieldLogger
	kv     core.KV
}

func New(injector *do.Injector) (*Backend, error) {
	return &Backend{
		docker: do.MustInvoke[*client.Client](injector),
		config: do.MustInvoke[core.Config](injector),
		logger: do.MustInvoke[logrus.FieldLogger](injector).WithField("component", "backends.dockerexternal.Backend"),
		kv:     do.MustInvoke[core.KV](injector),
	}, nil
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

	core.MapToEnvList(map[string]string{
		core.EnvNameFunctionName: function.Name(),
		core.EnvNameNatsURL:      b.config.NatsURL(),
	})

	containerConfig := &container.Config{
		Image: core.ImageNameRuntimeAPI,
		Env: core.MapToEnvList(map[string]string{
			core.EnvNameFunctionName: function.Name(),
			core.EnvNameNatsURL:      b.config.NatsURL(),
		}),
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

	containerConfig := &container.Config{
		Image: core.ImageNameRuntimeAPI,
		Env:   core.MapToEnvList(function.Env()),
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

func (b Backend) Info(ctx context.Context) (map[string]any, error) {
	info, err := b.docker.Info(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to docker info: %w", err)
	}

	return map[string]any{
		"backend":      "Docker backend",
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

	pod, err := CreateFunctionPod(ctx, function)
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
