package docker

import (
	"context"
	"fmt"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/google/uuid"
	"github.com/samber/do/v2"
	"github.com/zhulik/fid/internal/core"
)

const (
	APIDNSName = "api"
)

// FunctionPod is a struct that represents a group of a function instance and it's forwader living in the same network.
type FunctionPod struct {
	uuid   string // Of the "pod"
	config core.Config
	docker *client.Client

	runtimeAPIContainerName string
	functionContainerName   string
}

func CreateFunctionPod(
	ctx context.Context,
	function core.FunctionDefinition,
	injector do.Injector,
) (*FunctionPod, error) {
	podID := uuid.NewString()

	pod := &FunctionPod{
		uuid:                    podID,
		config:                  do.MustInvoke[core.Config](injector),
		docker:                  do.MustInvoke[*client.Client](injector),
		runtimeAPIContainerName: fmt.Sprintf("%s-%s", podID, core.ComponentNameRuntimeAPI),
		functionContainerName:   fmt.Sprintf("%s-%s", podID, core.ComponentNameFunction),
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

func (p FunctionPod) Stop(ctx context.Context) error {
	err := p.docker.ContainerStop(ctx, p.functionContainerName, container.StopOptions{})
	if err != nil {
		return fmt.Errorf("failed to stop container '%s': %w", p.functionContainerName, err)
	}

	err = p.docker.ContainerStop(ctx, p.runtimeAPIContainerName, container.StopOptions{})
	if err != nil {
		return fmt.Errorf("failed to stop container '%s': %w", p.runtimeAPIContainerName, err)
	}

	err = p.deleteNetwork(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (p FunctionPod) createNetwork(ctx context.Context) error {
	_, err := p.docker.NetworkCreate(ctx, p.uuid, network.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create network '%s': %w", p.uuid, err)
	}

	return nil
}

func (p FunctionPod) deleteNetwork(ctx context.Context) error {
	err := p.docker.NetworkRemove(ctx, p.uuid)
	if err != nil {
		return fmt.Errorf("failed to remove network '%s': %w", p.uuid, err)
	}

	return nil
}

func (p FunctionPod) createRuntimeAPI(ctx context.Context, function core.FunctionDefinition) error {
	containerConfig := &container.Config{
		Image: core.ImageNameFID,
		Cmd:   []string{core.ComponentNameRuntimeAPI},
		Env: core.MapToEnvList(map[string]string{
			core.EnvNameFunctionName:          function.Name(),
			core.EnvNameInstanceID:            p.uuid,
			core.EnvNameNatsURL:               p.config.NATSURL(),
			core.EnvNameFunctionContainerName: p.functionContainerName,
		}),
		Labels: map[string]string{
			core.LabelNameComponent: core.ComponentNameRuntimeAPI,
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
			p.uuid: {
				Aliases: []string{APIDNSName},
			},
			"nats": {}, // TODO: get this network name from somewhere else, remove hardcoded value
		},
	}

	resp, err := p.docker.ContainerCreate(
		ctx,
		containerConfig,
		hostConfig,
		networkingConfig,
		nil,
		p.runtimeAPIContainerName,
	)
	if err != nil {
		return fmt.Errorf("failed to create container: %w", err)
	}

	err = p.docker.ContainerStart(ctx, resp.ID, container.StartOptions{})
	if err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}

	return nil
}

func (p FunctionPod) createFunction(ctx context.Context, function core.FunctionDefinition) error {
	stopTimeout := int((function.Timeout() + time.Second) / time.Second)

	containerConfig := &container.Config{
		Image: function.Image(),
		Env: core.MapToEnvList(
			function.Env(),
			map[string]string{core.EnvNameAWSLambdaRuntimeAPI: APIDNSName},
		),
		Labels: map[string]string{
			core.LabelNameComponent: core.ComponentNameFunction,
		},
		StopTimeout: &stopTimeout,
	}
	hostConfig := &container.HostConfig{
		// AutoRemove: true,
	}
	networkingConfig := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			p.uuid: {},
		},
	}

	resp, err := p.docker.ContainerCreate(
		ctx,
		containerConfig,
		hostConfig,
		networkingConfig,
		nil,
		p.functionContainerName,
	)
	if err != nil {
		return fmt.Errorf("failed to create container: %w", err)
	}

	err = p.docker.ContainerStart(ctx, resp.ID, container.StartOptions{})
	if err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}

	return nil
}
