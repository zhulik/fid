package docker

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/zhulik/fid/internal/config"
	"github.com/zhulik/fid/internal/core"
	"github.com/zhulik/pal"
)

type Backend struct {
	Docker        *client.Client
	Config        config.Config
	Logger        *slog.Logger
	FunctionsRepo core.FunctionsRepo
	Pal           *pal.Pal
}

// Register creates a new function's template, scaler, and garbage collector(TODO).
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

// Deregister deletes function's template.
func (b Backend) Deregister(ctx context.Context, function core.FunctionDefinition) error {
	logger := b.Logger.With("function", function)

	err := b.FunctionsRepo.Delete(ctx, function.Name())
	if err != nil {
		return err //nolint:wrapcheck
	}

	// TODO: we only delete the definition, the containers should be stopped and deleted by the
	// garbage collector.

	logger.Info("Function deregistered")

	return nil
}

func (b Backend) createScaler(ctx context.Context, function core.FunctionDefinition) error {
	logger := b.Logger.With("function", function)

	containerConfig := &container.Config{
		Image: core.ImageNameFID,
		Cmd:   []string{core.ComponentNameScaler},
		Env: core.MapToEnvList(map[string]string{
			core.EnvNameFunctionName: function.Name(),
			core.EnvNameNatsURL:      b.Config.NATSURL,
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

	_, err := b.Docker.ContainerCreate(ctx, containerConfig, hostConfig, networkingConfig, nil, containerName)
	if err != nil {
		if strings.Contains(err.Error(), "Conflict. The container name") {
			logger.Info("Scaler container already exists")

			return nil
		}

		return fmt.Errorf("failed to create scaler container: %w", err)
	}

	err = b.Docker.ContainerStart(ctx, containerName, container.StartOptions{})
	if err != nil {
		return fmt.Errorf("failed to start scaler container: %w", err)
	}

	logger.Info("Scaler container created and started")

	return nil
}

func (b Backend) scalerContainerName(function core.FunctionDefinition) string {
	return fmt.Sprintf("%s-scaler", function)
}

func (b Backend) createFunctionTemplate(ctx context.Context, function core.FunctionDefinition) error {
	err := b.FunctionsRepo.Upsert(ctx, function)
	if err != nil {
		return fmt.Errorf("failed to store function template: %w", err)
	}

	b.Logger.With("function", function).Info("Function template stored")

	return nil
}

func (b Backend) Info(ctx context.Context) (map[string]any, error) {
	info, err := b.Docker.Info(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to docker info: %w", err)
	}

	return map[string]any{
		"backend":      "Docker backend",
		"dockerEngine": info,
	}, nil
}

func (b Backend) HealthCheck(ctx context.Context) error {
	b.Logger.Debug("ContainerBackend health check.")

	_, err := b.Docker.Info(ctx)
	if err != nil {
		return fmt.Errorf("backend health check failed: %w", err)
	}

	return nil
}

func (b Backend) Shutdown(_ context.Context) error {
	b.Logger.Debug("ContainerBackend shutting down...")
	defer b.Logger.Debug("ContainerBackend shot down.")

	err := b.Docker.Close()
	if err != nil {
		return fmt.Errorf("failed to shut down the backend: %w", err)
	}

	return nil
}

func (b Backend) AddInstance(ctx context.Context, function core.FunctionDefinition) (string, error) {
	b.Logger.Info("Creating new function pod", "function", function)

	pod := &FunctionPod{Function: function}

	err := b.Pal.InjectInto(ctx, pod)
	if err != nil {
		return "", fmt.Errorf("failed to inject dependencies into function pod: %w", err)
	}

	err = pod.Start(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to start function pod: %w", err)
	}

	b.Logger.Info("Function pod created", "function", function, "podID", pod.uuid)

	return pod.uuid, nil
}

func (b Backend) StopInstance(ctx context.Context, instanceID string) error {
	b.Logger.Info("Killing function instance", "instanceID", instanceID)

	return (&FunctionPod{uuid: instanceID, Docker: b.Docker}).Stop(ctx)
}

func (b Backend) StartGateway(ctx context.Context) (string, error) {
	containerConfig := &container.Config{
		Image: core.ImageNameFID,
		Cmd:   []string{core.ComponentNameGateway},
		Env: core.MapToEnvList(map[string]string{
			core.EnvNameNatsURL: b.Config.NATSURL,
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

	resp, err := b.Docker.ContainerCreate(
		ctx, containerConfig, hostConfig,
		networkingConfig, nil, core.ContainerNameGateway,
	)
	if err != nil {
		if strings.Contains(err.Error(), "Conflict. The container name") {
			b.Logger.Info("Gateway container already exists")

			return "", core.ErrContainerAlreadyExists
		}

		return "", fmt.Errorf("failed to create gateway container: %w", err)
	}

	err = b.Docker.ContainerStart(ctx, resp.ID, container.StartOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to start gateway container: %w", err)
	}

	b.Logger.Info("Gateway container created and started")

	return resp.ID, nil
}

func (b Backend) StartInfoServer(ctx context.Context) (string, error) {
	containerConfig := &container.Config{
		Image: core.ImageNameFID,
		Cmd:   []string{core.ComponentNameInfoServer},
		Env: core.MapToEnvList(map[string]string{
			core.EnvNameNatsURL: b.Config.NATSURL,
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

	resp, err := b.Docker.ContainerCreate(
		ctx, containerConfig, hostConfig,
		networkingConfig, nil, core.ContainerNameInfoServer,
	)
	if err != nil {
		if strings.Contains(err.Error(), "Conflict. The container name") {
			b.Logger.Info("Info server container already exists")

			return "", core.ErrContainerAlreadyExists
		}

		return "", fmt.Errorf("failed to create info server container: %w", err)
	}

	err = b.Docker.ContainerStart(ctx, resp.ID, container.StartOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to start info server container: %w", err)
	}

	b.Logger.Info("Info server container created and started")

	return resp.ID, nil
}
