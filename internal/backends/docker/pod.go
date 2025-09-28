package docker

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/google/uuid"
	"github.com/zhulik/fid/internal/config"
	"github.com/zhulik/fid/internal/core"
)

const (
	APIDNSName = "api"
)

// FunctionPod is a struct that represents a group of a function instance and it's runtime api
// living in the same network.
type FunctionPod struct {
	uuid string // Of the "pod"

	Config config.Config
	Docker *client.Client
	Logger *slog.Logger

	runtimeAPIContainerName string
	functionContainerName   string

	Function core.FunctionDefinition
}

func (p *FunctionPod) Init(ctx context.Context) error {
	p.Logger = p.Logger.With(
		"podID", p.uuid,
		"function", p.Function,
	)

	p.uuid = uuid.NewString()
	p.runtimeAPIContainerName = fmt.Sprintf("%s-%s", p.uuid, core.ComponentNameRuntimeAPI)
	p.functionContainerName = fmt.Sprintf("%s-%s", p.uuid, core.ComponentNameFunction)

	return nil
}

func (p *FunctionPod) Start(ctx context.Context) error {
	var err error

	defer func() {
		if err != nil {
			p.Logger.Warn("Pod creation failed, cleaning up...", "error", err)

			err := p.Stop(ctx)
			if err != nil {
				p.Logger.Warn("Failed to clean up after failed pod creation.", "error", err)
			}
		}
	}()

	_, err = p.Docker.NetworkCreate(ctx, p.uuid, network.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create network: %w", err)
	}

	err = p.createRuntimeAPI(ctx)
	if err != nil {
		return err
	}

	err = p.createFunction(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (p *FunctionPod) Stop(ctx context.Context) error {
	fnStopErr := p.Docker.ContainerStop(ctx, p.functionContainerName, container.StopOptions{})
	if fnStopErr != nil {
		if client.IsErrNotFound(fnStopErr) {
			p.Logger.Info("Function container '%s' does not exist, ignoring.")

			fnStopErr = nil
		} else {
			fnStopErr = fmt.Errorf("failed to stop container '%s': %w", p.functionContainerName, fnStopErr)
		}
	}

	apiStopErr := p.Docker.ContainerStop(ctx, p.runtimeAPIContainerName, container.StopOptions{})
	if apiStopErr != nil {
		if client.IsErrNotFound(apiStopErr) {
			p.Logger.Info("Runtime API container '%s' does not exist, ignoring.")

			fnStopErr = nil
		} else {
			apiStopErr = fmt.Errorf("failed to stop container '%s': %w", p.runtimeAPIContainerName, apiStopErr)
		}
	}

	netDeleteErr := p.Docker.NetworkRemove(ctx, p.uuid)
	if netDeleteErr != nil {
		netDeleteErr = fmt.Errorf("failed to delete network '%s': %w", p.uuid, netDeleteErr)
	}

	if fnStopErr != nil || apiStopErr != nil || netDeleteErr != nil {
		return errors.Join(fnStopErr, apiStopErr, netDeleteErr)
	}

	return nil
}

func (p *FunctionPod) createRuntimeAPI(ctx context.Context) error {
	containerConfig := &container.Config{
		Image: core.ImageNameFID,
		Cmd:   []string{core.ComponentNameRuntimeAPI},
		Env: core.MapToEnvList(map[string]string{
			core.EnvNameFunctionName:          p.Function.Name(),
			core.EnvNameInstanceID:            p.uuid,
			core.EnvNameNatsURL:               p.Config.NATSURL,
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

	resp, err := p.Docker.ContainerCreate(
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

	err = p.Docker.ContainerStart(ctx, resp.ID, container.StartOptions{})
	if err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}

	return nil
}

func (p *FunctionPod) createFunction(ctx context.Context) error {
	stopTimeout := int((p.Function.Timeout() + time.Second) / time.Second)

	containerConfig := &container.Config{
		Image: p.Function.Image(),
		Env: core.MapToEnvList(
			p.Function.Env(),
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

	resp, err := p.Docker.ContainerCreate(
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

	err = p.Docker.ContainerStart(ctx, resp.ID, container.StartOptions{})
	if err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}

	return nil
}
