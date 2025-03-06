package docker

import (
	"context"
	"fmt"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/samber/do"
	"github.com/sirupsen/logrus"
	"github.com/zhulik/fid/internal/core"
)

type Backend struct {
	docker        *client.Client
	config        core.Config
	logger        logrus.FieldLogger
	functionsRepo core.FunctionsRepo
}

func New(injector *do.Injector) (*Backend, error) {
	// TODO: define separate repositories for functions, elections etc.
	return &Backend{
		docker: do.MustInvoke[*client.Client](injector),
		config: do.MustInvoke[core.Config](injector),
		logger: do.MustInvoke[logrus.FieldLogger](injector).
			WithField("component", "backends.docker.Backend"),
		functionsRepo: do.MustInvoke[core.FunctionsRepo](injector),
	}, nil
}

// Register creates a new function's template, scaler, forwarder(TODO) and garbage collector(TODO).
func (b Backend) Register(ctx context.Context, function core.FunctionDefinition) error {
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

// Deregister deletes function's template, scaler, forwarder(TODO) and garbage collector(TODO).
func (b Backend) Deregister(ctx context.Context, function core.FunctionDefinition) error {
	// TODO: how to cleanup running instances?
	logger := b.logger.WithField("function", function)

	err := b.functionsRepo.Delete(ctx, function.Name())
	if err != nil {
		return err //nolint:wrapcheck
	}

	err = b.docker.ContainerStop(ctx, b.scalerContainerName(function), container.StopOptions{})
	if err != nil {
		return fmt.Errorf("failed to stop scaler: %w", err)
	}

	logger.Infof("Function deregistered")

	return nil
}

func (b Backend) createScaler(ctx context.Context, function core.FunctionDefinition) error {
	logger := b.logger.WithField("function", function)

	containerConfig := &container.Config{
		Image: core.ImageNameFID,
		Cmd:   []string{core.ComponentNameScaler},
		Env: core.MapToEnvList(map[string]string{
			core.EnvNameFunctionName: function.Name(),
			core.EnvNameNatsURL:      b.config.NATSURL(),
		}),
		Labels: map[string]string{
			core.LabelNameComponent: core.ComponentNameScaler,
		},
	}

	hostConfig := &container.HostConfig{
		Binds: []string{
			"/var/run/docker.sock:/var/run/docker.sock", // TODO: configurable
		},
		// AutoRemove: true,
	}
	networkingConfig := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			"nats": {}, // TODO: get from config
		},
	}

	containerName := b.scalerContainerName(function)

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

func (b Backend) scalerContainerName(function core.FunctionDefinition) string {
	return fmt.Sprintf("%s-scaler", function)
}

func (b Backend) createFunctionTemplate(ctx context.Context, function core.FunctionDefinition) error {
	err := b.functionsRepo.Upsert(ctx, function)
	if err != nil {
		return fmt.Errorf("failed to store function template: %w", err)
	}

	b.logger.WithField("function", function).Info("Function template stored")

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

func (b Backend) AddInstance(ctx context.Context, function core.FunctionDefinition) (string, error) {
	b.logger.Infof("Creating new function pod for function %s", function)

	pod, err := CreateFunctionPod(ctx, function)
	if err != nil {
		return "", err
	}

	b.logger.Infof("Function pod function %s created ID_=%s", function, pod.UUID)

	return pod.UUID, nil
}

func (b Backend) KillInstance(ctx context.Context, instanceID string) error {
	b.logger.Infof("Killing function instance %s", instanceID)

	return FunctionPod{UUID: instanceID, docker: b.docker}.Delete(ctx)
}

func (b Backend) StartGateway(ctx context.Context) (string, error) {
	containerConfig := &container.Config{
		Image: core.ImageNameFID,
		Cmd:   []string{core.ComponentNameGateway},
		Env: core.MapToEnvList(map[string]string{
			core.EnvNameNatsURL: b.config.NATSURL(),
		}),
		Labels: map[string]string{
			core.LabelNameComponent: core.ComponentNameGateway,
		},
		ExposedPorts: nat.PortSet{
			core.PortTCP80: struct{}{},
		},
	}

	hostConfig := &container.HostConfig{
		// AutoRemove: true,
		PortBindings: nat.PortMap{
			// TODO: configurable
			core.PortTCP80: {
				{
					HostPort: "8081",
					HostIP:   "0.0.0.0",
				},
			},
		},
	}
	networkingConfig := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			"nats": {}, // TODO: get from config
		},
	}

	resp, err := b.docker.ContainerCreate(
		ctx, containerConfig, hostConfig,
		networkingConfig, nil, core.ContainerNameGateway,
	)
	if err != nil {
		if strings.Contains(err.Error(), "Conflict. The container name") {
			b.logger.Infof("Gateway container already exists")

			return "", core.ErrContainerAlreadyExists
		}

		return "", fmt.Errorf("failed to create gateway container: %w", err)
	}

	err = b.docker.ContainerStart(ctx, resp.ID, container.StartOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to start gateway container: %w", err)
	}

	b.logger.Info("Gateway container created and started")

	return resp.ID, nil
}

func (b Backend) StartInfoServer(ctx context.Context) (string, error) {
	containerConfig := &container.Config{
		Image: core.ImageNameFID,
		Cmd:   []string{core.ComponentNameInfoServer},
		Env: core.MapToEnvList(map[string]string{
			core.EnvNameNatsURL: b.config.NATSURL(),
		}),
		Labels: map[string]string{
			core.LabelNameComponent: core.ComponentNameInfoServer,
		},
		ExposedPorts: nat.PortSet{
			core.PortTCP80: struct{}{},
		},
	}

	hostConfig := &container.HostConfig{
		Binds: []string{
			"/var/run/docker.sock:/var/run/docker.sock", // TODO: configurable
		},
		// AutoRemove: true,
		PortBindings: nat.PortMap{
			// TODO: configurable
			core.PortTCP80: {
				{
					HostPort: "8080",
					HostIP:   "0.0.0.0",
				},
			},
		},
	}
	networkingConfig := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			"nats": {}, // TODO: get from config
		},
	}

	resp, err := b.docker.ContainerCreate(
		ctx, containerConfig, hostConfig,
		networkingConfig, nil, core.ContainerNameInfoServer,
	)
	if err != nil {
		if strings.Contains(err.Error(), "Conflict. The container name") {
			b.logger.Infof("Info server container already exists")

			return "", core.ErrContainerAlreadyExists
		}

		return "", fmt.Errorf("failed to create info server container: %w", err)
	}

	err = b.docker.ContainerStart(ctx, resp.ID, container.StartOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to start info server container: %w", err)
	}

	b.logger.Info("Info server container created and started")

	return resp.ID, nil
}
