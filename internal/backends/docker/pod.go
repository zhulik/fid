package docker

import (
	"context"
	"fmt"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/google/uuid"
	"github.com/samber/do"
	"github.com/zhulik/fid/internal/core"
)

const (
	APIDNSName = "api"
)

// FunctionPod is a struct that represents a group of a function instance and it's forwader living in the same network.
type FunctionPod struct {
	UUID string // Of the "pod"

	docker *client.Client
}

func CreateFunctionPod(ctx context.Context, function core.Function) (*FunctionPod, error) {
	podID := uuid.NewString()

	pod := &FunctionPod{
		UUID:   podID,
		docker: do.MustInvoke[*client.Client](nil),
	}

	err := pod.createNetwork(ctx)
	if err != nil {
		return nil, err
	}

	err = pod.createRuntimeAPI(ctx, function)
	if err != nil {
		return nil, err
	}

	err = pod.createFunction(ctx, function)
	if err != nil {
		return nil, err
	}

	// TODO: cleanup on error

	return pod, nil
}

func (p FunctionPod) Delete(ctx context.Context) error {
	containerName := p.functionContainerName()

	err := p.docker.ContainerStop(ctx, containerName, container.StopOptions{})
	if err != nil {
		return fmt.Errorf("failed to stop container '%s': %w", containerName, err)
	}

	containerName = p.runtimeAPIContainerName()

	err = p.docker.ContainerStop(ctx, containerName, container.StopOptions{})
	if err != nil {
		return fmt.Errorf("failed to stop container '%s': %w", containerName, err)
	}

	err = p.deleteNetwork(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (p FunctionPod) createNetwork(ctx context.Context) error {
	_, err := p.docker.NetworkCreate(ctx, p.UUID, network.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create network '%s': %w", p.UUID, err)
	}

	return nil
}

func (p FunctionPod) deleteNetwork(ctx context.Context) error {
	err := p.docker.NetworkRemove(ctx, p.UUID)
	if err != nil {
		return fmt.Errorf("failed to remove network '%s': %w", p.UUID, err)
	}

	return nil
}

func (p FunctionPod) createRuntimeAPI(ctx context.Context, function core.Function) error {
	containerName := p.runtimeAPIContainerName()

	containerConfig := &container.Config{
		Image: core.ImageNameRuntimeAPI,
		Env: core.MapToEnvList(map[string]string{
			core.EnvNameFunctionName: function.Name(),
			core.EnvNameInstanceID:   p.UUID,
			// TODO: get this value from somewhere else, remove hardcoded value
			core.EnvNameNatsURL:               "nats://nats:4222",
			core.EnvNameFunctionContainerName: p.functionContainerName(),
		}),
		Labels: map[string]string{
			core.LabelNameComponent: core.RuntimeAPIComponentLabelValue,
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
			p.UUID: {
				Aliases: []string{APIDNSName},
			},
			"nats": {}, // TODO: get this network name from somewhere else, remove hardcoded value
		},
	}

	resp, err := p.docker.ContainerCreate(ctx, containerConfig, hostConfig, networkingConfig, nil, containerName)
	if err != nil {
		return fmt.Errorf("failed to create container: %w", err)
	}

	err = p.docker.ContainerStart(ctx, resp.ID, container.StartOptions{})
	if err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}

	return nil
}

func (p FunctionPod) createFunction(ctx context.Context, function core.Function) error {
	containerName := p.functionContainerName()

	stopTimeout := int((function.Timeout() + time.Second) / time.Second)

	containerConfig := &container.Config{
		Image: function.Image(),
		Env: core.MapToEnvList(
			function.Env(),
			map[string]string{core.EnvNameAWSLambdaRuntimeAPI: APIDNSName},
		),
		Labels: map[string]string{
			core.LabelNameComponent: core.FunctionComponentLabelValue,
		},
		StopTimeout: &stopTimeout,
	}
	hostConfig := &container.HostConfig{
		AutoRemove: true,
	}
	networkingConfig := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			p.UUID: {},
		},
	}

	resp, err := p.docker.ContainerCreate(ctx, containerConfig, hostConfig, networkingConfig, nil, containerName)
	if err != nil {
		return fmt.Errorf("failed to create container: %w", err)
	}

	err = p.docker.ContainerStart(ctx, resp.ID, container.StartOptions{})
	if err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}

	return nil
}

func (p FunctionPod) runtimeAPIContainerName() string {
	return p.UUID + "-runtimeapi"
}

func (p FunctionPod) functionContainerName() string {
	return p.UUID + "-function"
}
